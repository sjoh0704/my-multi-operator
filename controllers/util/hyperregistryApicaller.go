package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"

	coreV1 "k8s.io/api/core/v1"
)

type OidcConfig struct {
	AuthMode         string `json:"auth_mode,omitempty"`
	OidcAdminGroup   string `json:"oidc_admin_group,omitempty"`
	OidcAutoOnBoard  bool   `json:"oidc_auto_onboard,omitempty"`
	OidcClientId     string `json:"oidc_client_id,omitempty"`
	OidcClientSecret string `json:"oidc_client_secret,omitempty"`
	OidcEndpoint     string `json:"oidc_endpoint,omitempty"`
	OidcGroupsClaim  string `json:"oidc_groups_claim,omitempty"`
	OidcName         string `json:"oidc_name,omitempty"`
	OidcScope        string `json:"oidc_scope,omitempty"`
	OidcUserClaim    string `json:"oidc_user_claim,omitempty"`
	OidcVerifyCert   bool   `json:"oidc_verify_cert,omitempty"`
}

func SetHyperregistryServiceURI(hostpath string, serviceName string, urlParameter map[string]string) string {
	for key, value := range urlParameter {
		serviceName = strings.Replace(serviceName, "@@"+key+"@@", value, 1)
	}
	return "https://" + hostpath + serviceName
}

func SetHyperregistryOIDC(config OidcConfig, secret *coreV1.Secret, hostpath string) error {
	password := string(secret.Data["HARBOR_ADMIN_PASSWORD"])

	jsonData, err := json.Marshal(config)
	if err != nil {
		return err
	}
	playload := bytes.NewBuffer(jsonData)

	url := SetHyperregistryServiceURI(hostpath, HARBOR_SERVICE_SET_OIDC_CONFIG, nil)
	req, err := http.NewRequest(http.MethodPut, url, playload)
	if err != nil {
		return err
	}
	req.SetBasicAuth("admin", password)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if !IsOK(resp.StatusCode) {
		reqDump, _ := httputil.DumpRequest(req, true)
		return fmt.Errorf("failed to set oidc for hyperregistry: " + string(reqDump) + "\n" + resp.Status)
	}

	return nil
}

func IsOK(check int) bool {
	SuccessStatusList := map[int]bool{
		http.StatusOK:             true,
		http.StatusCreated:        true,
		http.StatusNoContent:      true,
		http.StatusPartialContent: true,
		// http.StatusConflict:       true,
		// http.StatusContinue:       true,
	}
	_, ok := SuccessStatusList[check]
	return ok
}
