package main

import (
	"context"
	"crypto/tls"
	"fmt"
	apiGrpc "github.com/awakari/subscriptions/api/grpc"
	apiGrpcAuth "github.com/awakari/subscriptions/api/grpc/auth"
	"github.com/awakari/subscriptions/api/grpc/events"
	"github.com/awakari/subscriptions/api/grpc/queue"
	apiGrpcPermits "github.com/awakari/subscriptions/api/grpc/usage/permits"
	apiHttp "github.com/awakari/subscriptions/api/http"
	apiHttpAuth "github.com/awakari/subscriptions/api/http/auth"
	"github.com/awakari/subscriptions/config"
	_ "github.com/awakari/subscriptions/docs"
	"github.com/awakari/subscriptions/service"
	"github.com/awakari/subscriptions/storage"
	"github.com/cloudevents/sdk-go/binding/format/protobuf/v2/pb"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/cache/v9"
	grpcpool "github.com/processout/grpc-go-pool"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log/slog"
	"net/http"
	"os"
)

// @title           Awakari Subscriptions API
// @version         1.0
// @description     Subscriptions API service is responsible for managing interest storage.

// @contact.name   Awakari Support
// @contact.email  awakari@awakari.com

// @BasePath  /

// @externalDocs.description  OpenAPI
// @externalDocs.url          https://swagger.io/resources/open-api/
func main() {

	cfg, err := config.NewConfigFromEnv()
	if err != nil {
		slog.Error(fmt.Sprintf("failed to load the config: %s", err))
	}
	opts := slog.HandlerOptions{
		Level: slog.Level(cfg.Log.Level),
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, &opts))
	log.Info("starting...")

	clientCache := redis.NewClient(&redis.Options{
		Addr:     cfg.Db.Cache.Addr,
		Password: cfg.Db.Cache.Password,
	})
	defer clientCache.Close()
	cacheCallbacks := cache.New(&cache.Options{
		Redis:      clientCache,
		LocalCache: cache.NewTinyLFU(int(cfg.Db.Cache.Local.Size), cfg.Db.Cache.Ttl),
	})

	var stor storage.Storage
	stor, err = storage.NewStorage(context.TODO(), cfg.Db)
	stor = storage.NewCache(stor, cacheCallbacks, cfg.Db.Cache.Ttl, log)
	if err != nil {
		panic(fmt.Sprintf("failed to initialize the callbacks storage: %s", err))
	}
	defer stor.Close()

	connPoolPermits, err := grpcpool.New(
		func() (*grpc.ClientConn, error) {
			return grpc.NewClient(cfg.Api.Usage.Uri, grpc.WithTransportCredentials(insecure.NewCredentials()))
		},
		int(cfg.Api.Usage.Connection.Count.Init),
		int(cfg.Api.Usage.Connection.Count.Max),
		cfg.Api.Usage.Connection.IdleTimeout,
	)
	if err != nil {
		panic(err)
	}
	defer connPoolPermits.Close()
	clientPermits := apiGrpcPermits.NewClientPool(connPoolPermits)
	svcPermits := apiGrpcPermits.NewService(clientPermits)
	svcPermits = apiGrpcPermits.NewServiceLogging(svcPermits, log)

	connPoolEvts, err := grpcpool.New(
		func() (*grpc.ClientConn, error) {
			return grpc.NewClient(cfg.Api.Events.Uri, grpc.WithTransportCredentials(insecure.NewCredentials()))
		},
		int(cfg.Api.Events.Connection.Count.Init),
		int(cfg.Api.Events.Connection.Count.Max),
		cfg.Api.Events.Connection.IdleTimeout,
	)
	if err != nil {
		panic(err)
	}
	defer connPoolEvts.Close()
	clientEvts := events.NewClientPool(connPoolEvts)
	svcEvts := events.NewService(clientEvts)
	svcEvts = events.NewLoggingMiddleware(svcEvts, log)
	err = svcEvts.SetStream(context.TODO(), cfg.Api.Events.FollowersChanged.Topic, cfg.Api.Events.FollowersChanged.Limit)
	if err != nil {
		panic(err)
	}

	countSubs := prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "awk_subscriptions_count",
			Help: "Awakari total subscriptions count",
		},
		func() (v float64) {
			var n int64
			n, err = stor.CountAll(context.Background())
			if err != nil {
				panic(err)
			}
			v = float64(n)
			return
		},
	)
	prometheus.MustRegister(countSubs)

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		http.ListenAndServe(fmt.Sprintf(":%d", cfg.Api.Metrics.Port), nil)
	}()
	//
	clientHttp := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	svc := service.NewService(
		stor,
		svcPermits,
		svcEvts,
		cfg.Api.Events,
	)
	svc = service.NewLogging(svc, log)

	if cfg.Api.Http.Enabled {

		var connAuth *grpc.ClientConn
		connAuth, err = grpc.NewClient(cfg.Api.Auth.Uri, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			panic(err)
		}
		clientAuth := apiGrpcAuth.NewServiceClient(connAuth)
		svcAuth := apiGrpcAuth.NewService(clientAuth)
		svcAuth = apiGrpcAuth.NewLogging(svcAuth, log)
		handlerAuth := apiHttpAuth.Handler{
			Svc: svcAuth,
		}

		r := gin.Default()
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))

		h := apiHttp.NewHandler(svc, clientHttp, cfg.Api.Http.UserAgent)
		r.
			Group("/v2", handlerAuth.Authorize).
			POST("", h.Update).
			GET("", h.Get)

		go r.Run(fmt.Sprintf(":%d", cfg.Api.Http.Port))
	}

	connPoolQueue, err := grpcpool.New(
		func() (*grpc.ClientConn, error) {
			return grpc.NewClient(cfg.Api.Queue.Uri, grpc.WithTransportCredentials(insecure.NewCredentials()))
		},
		int(cfg.Api.Queue.Connection.Count.Init),
		int(cfg.Api.Queue.Connection.Count.Max),
		cfg.Api.Queue.Connection.IdleTimeout,
	)
	if err != nil {
		panic(err)
	}
	defer connPoolQueue.Close()
	clientQueue := queue.NewClientPool(connPoolQueue)
	svcQueue := queue.NewService(clientQueue)
	svcQueue = queue.NewLoggingMiddleware(svcQueue, log)
	err = svcQueue.SetConsumer(context.TODO(), cfg.Api.Queue.InterestDeleted.Name, cfg.Api.Queue.InterestDeleted.Subj)
	log.Info(fmt.Sprintf("initialized the %s queue", cfg.Api.Queue.InterestDeleted.Name))
	if err != nil {
		panic(err)
	}
	go func() {
		err = consumeQueueInterestDeleted(context.Background(), svc, svcQueue, cfg.Api.Queue.InterestDeleted.Name, cfg.Api.Queue.InterestDeleted.Subj, cfg.Api.Queue.InterestDeleted.BatchSize)
		if err != nil {
			panic(err)
		}
	}()
	log.Info(fmt.Sprintf("started to consume the %s queue...", cfg.Api.Queue.InterestDeleted.Name))

	//go func() {
	//    time.Sleep(1 * time.Second)
	//    _ = svc.RestoreUsagePermits(context.Background())
	//}()

	log.Info(fmt.Sprintf("starting to listen the API @ port #%d...", cfg.Api.Port))
	if err = apiGrpc.Serve(svc, cfg.Api.Port); err != nil {
		panic(err)
	}
}

func consumeQueueInterestDeleted(ctx context.Context, svc service.Service, svcQueue queue.Service, name, subj string, batchSize uint32) (err error) {
	consume := func(evts []*pb.CloudEvent) (err error) {
		for _, e := range evts {
			interestId := e.GetTextData()
			_, err = svc.UnsubscribeByInterest(ctx, interestId)
		}
		return
	}
	for {
		err = svcQueue.ReceiveMessages(ctx, name, subj, batchSize, consume)
		if err != nil {
			break
		}
	}
	return
}
