package rfc2136

import (
	"context"
	"errors"

	rfc2136client "github.com/avarei/gardener-extension-dns-rfc2136/pkg/rfc2136/client"
	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	"github.com/miekg/dns"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewClientFromSecretRef creates a Client from a Secret Reference
func NewClientFromSecretRef(ctx context.Context, client client.Client, secretRef corev1.SecretReference) (*rfc2136client.Client, error) {
	creds, err := getCredentialsFromSecretRef(ctx, client, secretRef)
	if err != nil {
		return nil, err
	}

	return rfc2136client.NewClient(creds.TsigKeyName, creds.TsigSecret, creds.Alogrithm, creds.Server), nil
}

func getCredentialsFromSecretRef(ctx context.Context, client client.Client, secretRef corev1.SecretReference) (*Credentials, error) {
	secret, err := extensionscontroller.GetSecretByReference(ctx, client, &secretRef)
	if err != nil {
		return nil, err
	}
	creds := &Credentials{}
	tsigKeyName, ok := secret.Data["tsigKeyName"]
	if !ok {
		return nil, errors.New("missing tsigKeyName in secret")
	}
	creds.TsigKeyName = string(tsigKeyName)

	tsigSecret, ok := secret.Data["tsigSecret"]
	if !ok {
		return nil, errors.New("missing tsigSecret in secret")
	}
	creds.TsigSecret = string(tsigSecret)

	creds.Alogrithm = dns.HmacSHA256
	if algorithm, ok := secret.Data["algorithm"]; ok {
		creds.Alogrithm = string(algorithm)
	}

	if server, ok := secret.Data["server"]; ok {
		creds.Server = ptr.To(string(server))
	}

	return creds, nil
}
