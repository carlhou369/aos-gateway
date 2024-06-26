package rpc

import (
	"context"
	"fmt"
	chatapi "gateway/chat-api"
	"gateway/common"
	"gateway/db"
	"gateway/log"
	selfdriving "gateway/self-driving"
	"gateway/trie"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/goccy/go-json"
)

const SensitiveResponse = "作为一个人工智能，我无法对您上面提出的问题给出符合规范的、令您满意的回答，非常抱歉带给您糟糕的体验。感谢您提出的问题，我们后续会对此进行优化，以便能更好的为您服务。"

const InternalError = "internal server error"

const ClearCommandMsg = "clear"

const (
	LastMessageContextName      = "last_msg_id"
	LastConversationContextName = "last_conv_id"
	LastModelName               = "last_model_name"
	LastRelayUrlContextName     = "last_url"
	SesssionIdContextName       = "session_id"
)

const (
	Success            = 200
	ErrorCodeUnknow    = -500
	ErrorCodeReadReq   = -501
	ErrorCodeParseReq  = -502
	ErrorCodeUnmarshal = -503
)

const (
	HealthCheckUrl = "/health"
	QuestionUrl    = "/api"
)

var SessionCookieName = "session_id"

var Host = "127.0.0.1"

const (
	Avalible = 1
	InUse    = 2
	Down     = 0
)

const (
	CheckRelayHealthInterval = time.Second * 10
	WaitForAnswer            = time.Minute * 3
	MaxRetry                 = 0
)

var once sync.Once

var client *http.Client

var RpcServer *Service

type pendingQuestion struct {
	data       common.Question
	TriedTimes int
	resp       chan (RelayResponse)
	cancel     chan (struct{})
}

type Service struct {
	bsClientMut      sync.RWMutex
	port             string
	gptApiState      map[string]int                   //archived
	gptApiClients    map[string]*chatapi.Client       //api-key -> client
	bsApiClient      map[string][]*selfdriving.Client //modelname -> client
	urls             map[string]int
	relaysStateLock  sync.RWMutex
	questionCh       chan (pendingQuestion)
	maxPendingLength int
}

func InitRpcService(port string, relays []string, maxPendingLength int, bsModelConfig map[string][]string) {
	once.Do(func() {
		client = &http.Client{Timeout: time.Minute * 3}
		RpcServer = &Service{}
		RpcServer.port = port
		RpcServer.gptApiState = make(map[string]int)
		RpcServer.questionCh = make(chan pendingQuestion)
		RpcServer.maxPendingLength = maxPendingLength
		RpcServer.gptApiClients = make(map[string]*chatapi.Client)
		RpcServer.bsApiClient = make(map[string][]*selfdriving.Client)
		RpcServer.urls = make(map[string]int)
		RpcServer.relaysStateLock.Lock()
		defer RpcServer.relaysStateLock.Unlock()
		RpcServer.bsClientMut.Lock()
		defer RpcServer.bsClientMut.Unlock()
		for _, apiKey := range relays {
			RpcServer.gptApiState[apiKey] = Avalible
			RpcServer.gptApiClients[apiKey] = chatapi.NewClient(apiKey, context.Background())
		}
		for modelName, urls := range bsModelConfig {
			if RpcServer.bsApiClient[modelName] == nil {
				RpcServer.bsApiClient[modelName] = make([]*selfdriving.Client, 0)
			}
			for _, url := range urls {
				log.Info("init model name:", modelName, "url:", url)
				RpcServer.bsApiClient[modelName] = append(RpcServer.bsApiClient[modelName], selfdriving.NewClient(url, modelName, context.Background()))
				RpcServer.urls[url] = 1
			}
		}
	})
}

func (s *Service) checkHealth(ctx context.Context) {
	checkLoop := func() {
		for relay_url, _ := range s.gptApiState {
			if !strings.Contains(relay_url, "http") {
				continue
			}
			// log.Debug("check health %v", relay_url)
			res, err := get(relay_url + HealthCheckUrl)
			//fail check
			if err != nil || res == nil {
				RpcServer.relaysStateLock.Lock()
				log.Warn("relay down %v", relay_url)
				RpcServer.gptApiState[relay_url] = Down
				RpcServer.relaysStateLock.Unlock()
				continue
			}

			//pass check
			if RpcServer.gptApiState[relay_url] == Down {
				RpcServer.relaysStateLock.Lock()
				RpcServer.gptApiState[relay_url] = Avalible
				RpcServer.relaysStateLock.Unlock()
			}
		}
	}
	for {
		select {
		case <-ctx.Done():
			return
		default:
			checkLoop()
			time.Sleep(CheckRelayHealthInterval)
		}
	}
}

type LoggerMy struct {
}

func (*LoggerMy) Write(p []byte) (n int, err error) {
	msg := strings.TrimSpace(string(p))
	if strings.Index(msg, `"/healthcheck"`) > 0 {
		return
	}
	log.Debug(msg)
	return
}

func (c *Service) Start(ctx context.Context) error {
	postQuestionsContext, _ := context.WithCancel(ctx)
	go c.StartChatService(postQuestionsContext)

	//start gin
	gin.DefaultWriter = &LoggerMy{}
	r := gin.Default()
	//session middelware
	store := cookie.NewStore([]byte("secret11111")) //TODO:redis session store
	store.Options(sessions.Options{
		MaxAge: 20 * 60, //20min
	})
	//cors middleware
	r.Use(Cors())
	r.Use(sessions.Sessions("user", store))
	r.Use(UserSession())

	r.SetTrustedProxies(nil)
	r.GET("/healthcheck", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	r.POST("/api/fake", c.HandleFake)
	r.POST("/api/question", c.HandleQuestion)
	r.POST("/api/register", c.HandleRegister)
	r.POST("/api/register_worker", c.HandleRegisterWorker)
	r.POST("/api/receive_heart_beat", c.HandleSendHeartBeat)
	r.POST("/api/get_worker_address", c.HandleGetWorkerAddress)
	r.POST("/api/refresh_all_workers", c.HandleRefreshAllWorkers)
	r.POST("/api/list_language_models", c.HandleListModels)
	r.POST("/api/list_multimodal_models", c.HandleListMultiModals)
	r.POST("/api/worker_get_status", c.HandleWorkerGetStatus)
	r.GET("/api/refresh", func(c *gin.Context) {
		defer func() {
			c.String(http.StatusOK, "success")
		}()
		session_id := c.GetString(SesssionIdContextName)
		if session_id == "" {
			return
		}
		sess := sessions.Default(c)
		sess.Delete(session_id)
		sess.Save()
		log.Info("session id delete", session_id)
	})
	address := "0.0.0.0:" + c.port

	r.Run(address)
	log.Info("start rpc on port:" + c.port)
	return nil
}

type Resp struct {
	ResultCode int         `json:"ret"`
	ResultMsg  string      `json:"msg"`
	ResultBody interface{} `json:"data"`
}

type QuestionReq struct {
	Message        string `json:"message"`
	MessageId      string `json:"message_id"`
	ConversationId string `json:"conversation_id"`
	Model          string `json:"model"`
}

func (s *Service) HandleQuestion(c *gin.Context) {
	sess := sessions.Default(c)
	rep := Resp{
		ResultCode: ErrorCodeUnknow,
		ResultMsg:  "",
		ResultBody: "",
	}
	defer func() {
		if rep.ResultCode == Success {
			c.JSON(http.StatusOK, rep)
		} else {
			c.JSON(http.StatusInternalServerError, rep)
		}
	}()
	req := QuestionReq{}
	c.BindJSON(&req)
	//get session state
	msg := req.Message
	if msg == "" {
		log.Debug("no message")
		return
	}
	if msg == ClearCommandMsg {
		rep.ResultCode = Success
		data, _ := json.Marshal(&ProxyResponse{
			Text: "Cleared",
		})
		rep.ResultBody = string(data)
		session_id := c.GetString(SesssionIdContextName)
		if session_id == "" {
			return
		}
		sess := sessions.Default(c)
		sess.Delete(session_id)
		sess.Save()
		log.Info("session id delete", session_id)
		return
	}
	msg_id := req.MessageId
	conv_id := req.ConversationId
	modelName := req.Model
	s.bsClientMut.RLock()
	defer s.bsClientMut.RUnlock()
	if _, ok := s.bsApiClient[modelName]; !ok {
		if modelName != "" {
			rep.ResultMsg = fmt.Sprintf("model %s not supported", modelName)
			return
		}
		modelName = "gpt"
	}

	//check sensitive
	if trie.IsSensitive(msg) {
		log.Warn("sensitive message: ", msg)
		var data []byte
		data, _ = json.Marshal(&ProxyResponse{
			Text:           SensitiveResponse,
			MessageId:      "",
			ConversationId: "",
			Model:          modelName,
		})

		rep.ResultBody = string(data)
		rep.ResultCode = Success
		return
	}

	if modelName == c.GetString(LastModelName) {
		if msg_id == "" {
			msg_id = c.GetString(LastMessageContextName)
		}
		if conv_id == "" {
			conv_id = c.GetString(LastConversationContextName)
		}
	}

	sesson_id := c.GetString(SesssionIdContextName)
	q := common.Question{
		Message:        msg,
		MessageId:      msg_id,
		ConversationId: conv_id,
		OpenAIKey:      c.GetString(LastRelayUrlContextName),
		Model:          modelName,
	}
	qu := pendingQuestion{
		data:   q,
		resp:   make(chan RelayResponse, 1),
		cancel: make(chan struct{}),
	}
	defer close(qu.cancel)

	timer := time.NewTimer(WaitForAnswer)

	select {
	case <-timer.C:
		log.Warn(fmt.Sprintf("pending question time out %v", q))
		return
	case s.questionCh <- qu:
		select {
		case <-timer.C:
			sess.Delete(sesson_id)
			sess.Save()
			return
		case answer := <-qu.resp:
			if answer.Text == "" {
				return
			}

			go func() {
				err := db.InsertSingleConversation(db.Message{
					ConversationId: answer.ConversationId,
					MessageId:      answer.MessageId,
					Prompt:         q.Message,
					Text:           answer.Text,
					StartTime:      time.Now().Unix(),
					Model:          answer.Model,
					Url:            answer.Url,
				})
				if err != nil {
					log.Error("insert into db error", err)
				}
			}()

			var data []byte
			data, _ = json.Marshal(&ProxyResponse{
				Text:           answer.Text,
				MessageId:      answer.MessageId,
				ConversationId: answer.ConversationId,
				Model:          answer.Model,
			})

			rep.ResultBody = string(data)
			rep.ResultCode = Success
			//update session
			if sesson_id != "" {
				if answer.Text == InternalError || answer.Text == "" {
					sess.Delete(sesson_id)
					sess.Save()
					return
				}
				modelName := "gpt"
				if _, ok := s.bsApiClient[answer.Model]; ok {
					modelName = answer.Model
				}
				data, _ := json.Marshal(&UserStatus{
					ConversationId: answer.ConversationId,
					MessageId:      answer.MessageId,
					Url:            answer.Url,
					LastTime:       time.Now().Unix(),
					Model:          modelName,
				})
				sess.Set(sesson_id, string(data))
				sess.Save()
			}
			return
		}
	}
}

type WorkerRegReq struct {
	Model string `json:"model"`
	Url   string `json:"url"`
}

func (s *Service) HandleRegister(c *gin.Context) {
	rep := Resp{
		ResultCode: 200,
		ResultMsg:  "",
		ResultBody: "",
	}
	defer func() {
		if rep.ResultCode == Success {
			c.JSON(http.StatusOK, rep)
		} else {
			c.JSON(rep.ResultCode, rep)
		}
	}()
	req := WorkerRegReq{}
	c.BindJSON(&req)
	model := req.Model
	url := req.Url
	s.bsClientMut.Lock()
	defer s.bsClientMut.Unlock()
	if _, ok := s.bsApiClient[model]; !ok {
		rep.ResultCode = 403
		rep.ResultMsg = "model not supported yet"
		return
	}
	if _, ok := s.urls[url]; ok {
		rep.ResultCode = 403
		rep.ResultMsg = "already regitered"
		return
	}
	s.urls[url] = 1
	s.bsApiClient[model] = append(s.bsApiClient[model], selfdriving.NewClient(url, model, context.TODO()))
	rep.ResultMsg = "ok"
}

type FastChatWorkerStatus struct {
	ModelNames  []string `json:"model_names"`
	Speed       int      `json:"speed"`
	QueueLength int      `json:"queue_length"`
}

type FastChatRegisterWorkerReq struct {
	WorkerName     string               `json:"worker_name"`
	CheckHeartBeat bool                 `json:"check_heart_beat"`
	WorkerStatus   FastChatWorkerStatus `json:"worker_status"`
	MultiModal     bool                 `json:"multimodal"`
}

func (s *Service) HandleRegisterWorker(c *gin.Context) {
	rep := Resp{
		ResultCode: 200,
		ResultMsg:  "",
		ResultBody: "",
	}
	defer func() {
		if rep.ResultCode == Success {
			c.JSON(http.StatusOK, rep)
		} else {
			c.JSON(http.StatusInternalServerError, rep)
		}
	}()
	req := FastChatRegisterWorkerReq{}
	c.BindJSON(&req)
	s.bsClientMut.Lock()
	defer s.bsClientMut.Unlock()
	for _, model := range req.WorkerStatus.ModelNames {
		url := req.WorkerName
		if _, ok := s.bsApiClient[model]; !ok {
			rep.ResultCode = 403
			rep.ResultMsg = "model not supported yet"
			return
		}
		if _, ok := s.urls[url]; ok {
			continue
		}
		s.bsApiClient[model] = append(s.bsApiClient[model], selfdriving.NewClient(url, model, context.TODO()))
	}
	rep.ResultMsg = "ok"
}

type SendHeartBeatReq struct {
	WorkerName  string `json:"worker_name"`
	QueueLength int    `json:"queue_length"`
}

type SendHeartBeatResp struct {
	Exist bool `json:"exist"`
}

func (s *Service) HandleSendHeartBeat(c *gin.Context) {
	req := SendHeartBeatReq{}
	c.BindJSON(&req)
	_, exist := s.urls[req.WorkerName]
	c.JSON(http.StatusOK, SendHeartBeatResp{exist})
}

type GetWorkerAddressReq struct {
	Model string `json:"model"`
}

type GetWorkerAddressResp struct {
	Address string `json:"address"`
}

func (s *Service) HandleGetWorkerAddress(c *gin.Context) {
	req := GetWorkerAddressReq{}
	c.BindJSON(&req)
	s.bsClientMut.RLock()
	defer s.bsClientMut.RUnlock()
	addr := ""
	bsClients, ok := s.bsApiClient[req.Model]
	if !ok {
		c.JSON(http.StatusOK, GetWorkerAddressResp{})
		return
	}
	for _, cli := range bsClients {
		if cli.Status == selfdriving.ModelAvalible {
			addr = cli.Url
			break
		}
	}
	c.JSON(http.StatusOK, GetWorkerAddressResp{addr})
}

func (s *Service) HandleRefreshAllWorkers(c *gin.Context) {
	c.JSON(http.StatusOK, "")
}

type ListModelsResp struct {
	Models []string `json:"models"`
}

func (s *Service) HandleListModels(c *gin.Context) {
	s.bsClientMut.RLock()
	defer s.bsClientMut.RUnlock()
	models := make([]string, 0)
	for model, _ := range s.bsApiClient {
		models = append(models, model)
	}
	c.JSON(http.StatusOK, ListModelsResp{models})
}

func (s *Service) HandleListMultiModals(c *gin.Context) {
	c.JSON(http.StatusOK, ListModelsResp{[]string{}})
}

func (s *Service) HandleWorkerGetStatus(c *gin.Context) {
	rep := Resp{
		ResultCode: 200,
		ResultMsg:  "",
		ResultBody: "",
	}
	c.JSON(http.StatusOK, rep)
}

func (s *Service) HandleFake(c *gin.Context) {
	rep := Resp{
		ResultCode: 200,
		ResultMsg:  "",
		ResultBody: "",
	}
	defer func() {
		if rep.ResultCode == Success {
			c.JSON(http.StatusOK, rep)
		} else {
			c.JSON(http.StatusInternalServerError, rep)
		}
	}()
	pr := ProxyResponse{
		Text:           "fake",
		MessageId:      "fake",
		ConversationId: "fake",
	}
	data, _ := json.Marshal(&pr)
	rep.ResultBody = string(data)
}

func (s *Service) StartChatService(ctx context.Context) {
	//max concurrent questions
	handling := make(chan struct{}, s.maxPendingLength)
	for {
		select {
		case <-ctx.Done():
			return
		case qu := <-s.questionCh:
			handling <- struct{}{}
			log.Debug("try send question to relay ", qu.data)
			go func() {
				s.checkOneQuestion(qu)
				log.Debug("question to relay done ", qu.data)
				<-handling
			}()
		}
	}
}

func (s *Service) checkOneQuestion(qu pendingQuestion) {
	//retry a MaxRetry times
	for {
		select {
		case <-qu.cancel:
			log.Info("close for timeout %v", qu.data)
			return
		default:
			if qu.TriedTimes > MaxRetry {
				close(qu.resp)
				return
			}
			if qu.TriedTimes != 0 {
				log.Debug("retry question:", qu.data)
			}
			s.bsClientMut.RLock()
			defer s.bsClientMut.RUnlock()
			if clients, ok := s.bsApiClient[qu.data.Model]; ok && len(clients) != 0 {
				if toRetry := s.handleBsQuestion(qu); toRetry {
					jitter := rand.Intn(500)
					time.Sleep(time.Millisecond * (1000 + time.Duration(jitter)))
					continue
				}
				return
			}
			//gpt3 model
			if toRetry := s.handleOpenAIQuestion(qu); toRetry != nil {
				jitter := rand.Intn(500)
				time.Sleep(time.Millisecond * (1000 + time.Duration(jitter)))
				continue
			}
			return
		}
	}
}
