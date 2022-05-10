// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package nonk8sapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"github.com/stolostron/hub-of-hubs-manager/pkg/nonk8sapi/authentication"
	"github.com/stolostron/hub-of-hubs-manager/pkg/nonk8sapi/managedclusters"
	"github.com/stolostron/hub-of-hubs-manager/pkg/specsyncer/db2transport/db"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	secondsToFinishOnShutdown = 5
)

// nonK8sApiServer defines the non-k8s-api-server
type nonK8sApiServer struct {
	log             logr.Logger
	certificateFile string
	keyFile         string
	svr             *http.Server
}

// AddNonK8sApiServer adds the non-k8s-api-server to the Manager.
func AddNonK8sApiServer(mgr ctrl.Manager, clusterAPIURL, authorizationURL, basePath, certificateFile, keyFile string, clusterAPICABundle, authorizationCABundle []byte, database db.DB) error {
	router := gin.Default()
	router.Use(authentication.Authentication(clusterAPIURL, clusterAPICABundle))

	routerGroup := router.Group(basePath)
	routerGroup.GET("/managedclusters", managedclusters.List(authorizationURL, authorizationCABundle, database.GetConn()))

	routerGroup.PATCH("/managedclusters/:cluster", managedclusters.Patch(authorizationURL, authorizationCABundle, database.GetConn()))

	err := mgr.Add(&nonK8sApiServer{
		log: ctrl.Log.WithName("non-k8s-api-server"),
		svr: &http.Server{
			Addr:    ":8080",
			Handler: router,
		},
		certificateFile: certificateFile,
		keyFile:         keyFile,
	})
	if err != nil {
		return fmt.Errorf("failed to add non k8s api server to the manager: %w", err)
	}

	return nil
}

// Start runs the non-k8s-api server within given context
func (s *nonK8sApiServer) Start(ctx context.Context) error {
	idleConnsClosed := make(chan struct{})
	// initializing the shutdown process in a goroutine so that it won't block the server starting and running
	go func() {
		<-ctx.Done()
		s.log.Info("shutting down non-k8s-api server")

		// The context is used to inform the server it has 5 seconds to finish the request it is currently handling
		shutdownCtx, cancel := context.WithTimeout(context.Background(), secondsToFinishOnShutdown*time.Second)
		defer cancel()
		if err := s.svr.Shutdown(shutdownCtx); err != nil {
			// Error from closing listeners, or context timeout
			s.log.Error(err, "error shutting down the non-k8s-api server")
		}

		s.log.Info("the non-k8s-api server is exiting")
		close(idleConnsClosed)
	}()

	if err := s.svr.ListenAndServeTLS(s.certificateFile, s.keyFile); err != nil && errors.Is(err, http.ErrServerClosed) {
		return err
	}

	<-idleConnsClosed
	return nil
}
