package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/blacktop/xpost/internal/xpost"
	"github.com/spf13/cobra"
)

func newBridgeCommand() *cobra.Command {
	return &cobra.Command{
		Use:           "bridge",
		Short:         "Publish one structured post",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE:          runBridge,
	}
}

func runBridge(cmd *cobra.Command, _ []string) error {
	var request xpost.BridgeRequest
	decoder := json.NewDecoder(cmd.InOrStdin())
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		return fmt.Errorf("decode bridge request: %w", err)
	}

	var extra json.RawMessage
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("decode bridge request: multiple JSON values")
		}
		return fmt.Errorf("decode bridge request: %w", err)
	}

	if request.Operation != xpost.BridgeOperationPublish {
		return fmt.Errorf("unsupported bridge operation %q", request.Operation)
	}
	target, err := normalizeBridgeTarget(request.Target)
	if err != nil {
		return writeBridgeResponse(cmd, xpost.BridgeResponse{
			Status:    xpost.BridgeStatusRejected,
			Error:     err.Error(),
			ErrorKind: "validation",
		})
	}

	configuration, err := loadConfiguration(configPath)
	if err != nil {
		return writeBridgeResponse(cmd, bridgeErrorResponse(err))
	}

	posterList, err := buildPosters(cmd.Context(), []string{target}, configuration)
	if err != nil {
		return writeBridgeResponse(cmd, bridgeErrorResponse(err))
	}
	if len(posterList) != 1 {
		return errors.New("bridge target did not produce one poster")
	}

	requestPost := xpost.Request{
		Message:     request.Text,
		Attachments: request.Attachments,
		ReplyTo:     request.ReplyTo,
		RootReplyTo: request.RootReplyTo,
	}
	poster := posterList[0]
	if err := poster.Validate(requestPost); err != nil {
		return writeBridgeResponse(cmd, bridgeErrorResponse(err))
	}

	resultPoster, ok := poster.(xpost.ResultPoster)
	if !ok {
		return writeBridgeResponse(cmd, xpost.BridgeResponse{
			Status:    xpost.BridgeStatusFailed,
			Error:     "poster does not return a publication result",
			ErrorKind: "protocol",
		})
	}
	result, err := resultPoster.PostResult(cmd.Context(), requestPost)
	if err != nil {
		return writeBridgeResponse(cmd, bridgeErrorResponse(err))
	}

	return writeBridgeResponse(cmd, xpost.BridgeResponse{
		Status:    xpost.BridgeStatusPublished,
		RemoteID:  result.RemoteID,
		RemoteCID: result.RemoteCID,
		URL:       result.URL,
	})
}

func normalizeBridgeTarget(target string) (string, error) {
	target = strings.ToLower(strings.TrimSpace(target))
	if target == "x" {
		return "twitter", nil
	}
	if _, ok := supportedTargets[target]; !ok || target == "x" {
		return "", fmt.Errorf("unsupported target %q", target)
	}
	return target, nil
}

func bridgeErrorResponse(err error) xpost.BridgeResponse {
	status := xpost.BridgeStatusFailed
	kind := "transport"
	var validationErr xpost.ValidationError
	var missingEnvErr xpost.MissingEnvError
	var configurationErr configurationError
	var configLoadErr configLoadError
	if errors.As(err, &validationErr) || errors.As(err, &missingEnvErr) ||
		errors.As(err, &configurationErr) || errors.As(err, &configLoadErr) {
		status = xpost.BridgeStatusRejected
		kind = "validation"
		if errors.As(err, &configurationErr) || errors.As(err, &configLoadErr) {
			kind = "configuration"
		}
	}
	return xpost.BridgeResponse{
		Status:    status,
		Error:     err.Error(),
		ErrorKind: kind,
	}
}

func writeBridgeResponse(cmd *cobra.Command, response xpost.BridgeResponse) error {
	if err := json.NewEncoder(cmd.OutOrStdout()).Encode(response); err != nil {
		return fmt.Errorf("encode bridge response: %w", err)
	}
	return nil
}
