// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2020-2021 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

package http

import (
	"encoding/json"
	"net/http"
	"strings"

	bootstrapContainer "github.com/edgexfoundry/go-mod-bootstrap/v3/bootstrap/container"
	"github.com/edgexfoundry/go-mod-bootstrap/v3/bootstrap/interfaces"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/common"
	commonDTO "github.com/edgexfoundry/go-mod-core-contracts/v3/dtos/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/errors"

	sdkCommon "github.com/edgexfoundry/device-sdk-go/v3/internal/common"
	"github.com/edgexfoundry/device-sdk-go/v3/internal/config"
	"github.com/edgexfoundry/device-sdk-go/v3/internal/container"
)

// Ping handles the request to /ping endpoint. Is used to test if the service is working
// It returns a response as specified by the V2 API swagger in openapi/common
func (c *RestController) Ping(writer http.ResponseWriter, request *http.Request) {
	response := commonDTO.NewPingResponse(c.serviceName)
	c.sendResponse(writer, request, common.ApiPingRoute, response, http.StatusOK)
}

// Version handles the request to /version endpoint. Is used to request the service's versions
// It returns a response as specified by the V2 API swagger in openapi/common
func (c *RestController) Version(writer http.ResponseWriter, request *http.Request) {
	response := commonDTO.NewVersionSdkResponse(sdkCommon.ServiceVersion, sdkCommon.SDKVersion, c.serviceName)
	c.sendResponse(writer, request, common.ApiVersionRoute, response, http.StatusOK)
}

// Config handles the request to /config endpoint. Is used to request the service's configuration
// It returns a response as specified by the V2 API swagger in openapi/common
func (c *RestController) Config(writer http.ResponseWriter, request *http.Request) {
	var fullConfig interface{}
	configuration := container.ConfigurationFrom(c.dic.Get)

	if c.customConfig == nil {
		// case of no custom configs
		fullConfig = *configuration
	} else {
		// create a struct combining the common configuration and custom configuration sections
		fullConfig = struct {
			config.ConfigurationStruct
			CustomConfiguration interfaces.UpdatableConfig
		}{
			*configuration,
			c.customConfig,
		}
	}

	response := commonDTO.NewConfigResponse(fullConfig, c.serviceName)
	c.sendResponse(writer, request, common.ApiVersionRoute, response, http.StatusOK)
}

// Secret handles the request to add Device Service exclusive secret to the Secret Store
// It returns a response as specified by the V2 API swagger in openapi/common
func (c *RestController) Secret(writer http.ResponseWriter, request *http.Request) {
	defer func() {
		_ = request.Body.Close()
	}()

	provider := bootstrapContainer.SecretProviderFrom(c.dic.Get)
	secretRequest := commonDTO.SecretRequest{}
	err := json.NewDecoder(request.Body).Decode(&secretRequest)
	if err != nil {
		edgexError := errors.NewCommonEdgeX(errors.KindContractInvalid, "JSON decode failed", err)
		c.sendEdgexError(writer, request, edgexError, common.ApiSecretRoute)
		return
	}

	path, secret := c.prepareSecret(secretRequest)

	if err := provider.StoreSecret(path, secret); err != nil {
		edgexError := errors.NewCommonEdgeX(errors.KindServerError, "Storing secret failed", err)
		c.sendEdgexError(writer, request, edgexError, common.ApiSecretRoute)
		return
	}

	response := commonDTO.NewBaseResponse(secretRequest.RequestId, "", http.StatusCreated)
	c.sendResponse(writer, request, common.ApiSecretRoute, response, http.StatusCreated)
}

func (c *RestController) prepareSecret(request commonDTO.SecretRequest) (string, map[string]string) {
	var secretKVs = make(map[string]string)
	for _, secret := range request.SecretData {
		secretKVs[secret.Key] = secret.Value
	}

	path := strings.TrimSpace(request.SecretName)

	return path, secretKVs
}
