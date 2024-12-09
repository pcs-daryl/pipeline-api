package middleware

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pcs-aa-aas/commons/pkg/api/dto"
	"github.com/pcs-aa-aas/commons/pkg/log"
	"k8s.io/client-go/tools/clientcmd/api"
)

func KubeconfigMiddleware(kubeconfigUrl string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(kubeconfigUrl) < 1 {
			log.Fatalf("no kubeconfig url, configure the kubeconfig url")
		}

		token := c.GetHeader("Authorization")
		if len(token) < 1 {
			errMsg := "error: missing bearer token"
			log.Errorf(errMsg)
			err := errors.New(errMsg)
			c.AbortWithError(http.StatusUnauthorized, err)
			return
		}

		email := c.Request.Header.Get("x-jwt-email")
		if len(email) < 1 {
			errMsg := "error: missing jwt-email"
			log.Errorf(errMsg)
			err := errors.New(errMsg)
			c.AbortWithError(http.StatusUnauthorized, err)
			return
		}

		targetCluster := c.Request.Header.Get("X-Target-Cluster")
		if len(targetCluster) < 1 {
			errMsg := "error: missing target cluster"
			log.Errorf(errMsg)
			err := errors.New(errMsg)
			c.AbortWithError(http.StatusUnauthorized, err)
			return
		}

		// TODO - enable MTLS and use https & also use svc-espresso Service instead of Ingress Gateway
		espressoUrl := "http://portal.mocha.dev.devnet3.staging/espresso/v1/kubeconfig/object?target_cluster=" + targetCluster
		// TODO - to use espressoHost, to be passed from config on start up. To cater for different environments
		// espressoUrl := kubeconfigUrl + targetContext
		httpClient := &http.Client{}
		req, _ := http.NewRequest("GET", espressoUrl, nil)
		req.Header.Add("Authorization", token)
		req.Header.Add("x-jwt-email", email)

		resp, err := httpClient.Do(req)
		if err != nil {
			log.Errorf("Error fetching data from espresso: %v", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		defer resp.Body.Close()

		var response dto.ResponseMsg
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			log.Errorf("Error decoding response from espresso due to: %v", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		// Marshal kubeconfig (interface{}) to a json
		jsonData, err := json.Marshal(response.Result)
		if err != nil {
			log.Errorf("Error marshalling response results due to: %v", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		kubeconfig := &api.Config{}
		err = json.Unmarshal(jsonData, kubeconfig)
		if err != nil {
			log.Errorf("Error unmarshalling kubeconfig due to: %v", err)
			c.AbortWithError(http.StatusInternalServerError, err)
			return
		}

		c.Set("kubeconfig", kubeconfig)
		c.Next()
	}
}
