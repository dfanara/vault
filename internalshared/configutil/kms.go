// +build !enterprise

package configutil

import (
	"crypto/rand"
	"fmt"
	"io"

	"github.com/hashicorp/errwrap"
	"github.com/hashicorp/go-hclog"
	wrapping "github.com/hashicorp/go-kms-wrapping"
	aeadwrapper "github.com/hashicorp/go-kms-wrapping/wrappers/aead"
	"github.com/hashicorp/go-kms-wrapping/wrappers/alicloudkms"
	"github.com/hashicorp/go-kms-wrapping/wrappers/awskms"
	"github.com/hashicorp/go-kms-wrapping/wrappers/azurekeyvault"
	"github.com/hashicorp/go-kms-wrapping/wrappers/gcpckms"
	"github.com/hashicorp/go-kms-wrapping/wrappers/ocikms"
	"github.com/hashicorp/go-kms-wrapping/wrappers/transit"
	"github.com/hashicorp/vault/sdk/logical"
)

var (
	ConfigureWrapper             = configureWrapper
	CreateSecureRandomReaderFunc = createSecureRandomReader
)

func configureWrapper(configKMS *KMS, infoKeys *[]string, info *map[string]string, logger hclog.Logger) (wrapping.Wrapper, error) {
	var wrapper wrapping.Wrapper
	var kmsInfo map[string]string
	var err error

	opts := &wrapping.WrapperOptions{
		Logger: logger,
	}

	switch configKMS.Type {
	case wrapping.Shamir:
		return nil, nil

	case wrapping.AEAD:
		wrapper, kmsInfo, err = GetAEADKMSFunc(opts, configKMS)

	case wrapping.AliCloudKMS:
		wrapper, kmsInfo, err = GetAliCloudKMSFunc(opts, configKMS)

	case wrapping.AWSKMS:
		wrapper, kmsInfo, err = GetAWSKMSFunc(opts, configKMS)

	case wrapping.AzureKeyVault:
		wrapper, kmsInfo, err = GetAzureKeyVaultKMSFunc(opts, configKMS)

	case wrapping.GCPCKMS:
		wrapper, kmsInfo, err = GetGCPCKMSKMSFunc(opts, configKMS)

	case wrapping.OCIKMS:
		wrapper, kmsInfo, err = GetOCIKMSKMSFunc(opts, configKMS)

	case wrapping.Transit:
		wrapper, kmsInfo, err = GetTransitKMSFunc(opts, configKMS)

	case wrapping.PKCS11:
		return nil, fmt.Errorf("KMS type 'pkcs11' requires the Vault Enterprise HSM binary")

	default:
		return nil, fmt.Errorf("Unknown KMS type %q", configKMS.Type)
	}

	if err != nil {
		return nil, err
	}

	for k, v := range kmsInfo {
		*infoKeys = append(*infoKeys, k)
		(*info)[k] = v
	}

	return wrapper, nil
}

var GetAEADKMSFunc = func(opts *wrapping.WrapperOptions, kms *KMS) (wrapping.Wrapper, map[string]string, error) {
	wrapper := aeadwrapper.NewWrapper(opts)
	wrapperInfo, err := wrapper.SetConfig(kms.Config)
	if err != nil {
		return nil, nil, err
	}
	info := make(map[string]string)
	if wrapperInfo != nil {
		str := "AEAD Type"
		if kms.Purpose != "" {
			str = fmt.Sprintf("%s %s", kms.Purpose, str)
		}
		info[str] = wrapperInfo["aead_type"]
	}
	return wrapper, info, nil
}

var GetAliCloudKMSFunc = func(opts *wrapping.WrapperOptions, kms *KMS) (wrapping.Wrapper, map[string]string, error) {
	wrapper := alicloudkms.NewWrapper(opts)
	wrapperInfo, err := wrapper.SetConfig(kms.Config)
	if err != nil {
		// If the error is any other than logical.KeyNotFoundError, return the error
		if !errwrap.ContainsType(err, new(logical.KeyNotFoundError)) {
			return nil, nil, err
		}
	}
	info := make(map[string]string)
	if wrapperInfo != nil {
		info["AliCloud KMS Region"] = wrapperInfo["region"]
		info["AliCloud KMS KeyID"] = wrapperInfo["kms_key_id"]
		if domain, ok := wrapperInfo["domain"]; ok {
			info["AliCloud KMS Domain"] = domain
		}
	}
	return wrapper, info, nil
}

var GetAWSKMSFunc = func(opts *wrapping.WrapperOptions, kms *KMS) (wrapping.Wrapper, map[string]string, error) {
	wrapper := awskms.NewWrapper(opts)
	wrapperInfo, err := wrapper.SetConfig(kms.Config)
	if err != nil {
		// If the error is any other than logical.KeyNotFoundError, return the error
		if !errwrap.ContainsType(err, new(logical.KeyNotFoundError)) {
			return nil, nil, err
		}
	}
	info := make(map[string]string)
	if wrapperInfo != nil {
		info["AWS KMS Region"] = wrapperInfo["region"]
		info["AWS KMS KeyID"] = wrapperInfo["kms_key_id"]
		if endpoint, ok := wrapperInfo["endpoint"]; ok {
			info["AWS KMS Endpoint"] = endpoint
		}
	}
	return wrapper, info, nil
}

var GetAzureKeyVaultKMSFunc = func(opts *wrapping.WrapperOptions, kms *KMS) (wrapping.Wrapper, map[string]string, error) {
	wrapper := azurekeyvault.NewWrapper(opts)
	wrapperInfo, err := wrapper.SetConfig(kms.Config)
	if err != nil {
		// If the error is any other than logical.KeyNotFoundError, return the error
		if !errwrap.ContainsType(err, new(logical.KeyNotFoundError)) {
			return nil, nil, err
		}
	}
	info := make(map[string]string)
	if wrapperInfo != nil {
		info["Azure Environment"] = wrapperInfo["environment"]
		info["Azure Vault Name"] = wrapperInfo["vault_name"]
		info["Azure Key Name"] = wrapperInfo["key_name"]
	}
	return wrapper, info, nil
}

var GetGCPCKMSKMSFunc = func(opts *wrapping.WrapperOptions, kms *KMS) (wrapping.Wrapper, map[string]string, error) {
	wrapper := gcpckms.NewWrapper(opts)
	wrapperInfo, err := wrapper.SetConfig(kms.Config)
	if err != nil {
		// If the error is any other than logical.KeyNotFoundError, return the error
		if !errwrap.ContainsType(err, new(logical.KeyNotFoundError)) {
			return nil, nil, err
		}
	}
	info := make(map[string]string)
	if wrapperInfo != nil {
		info["GCP KMS Project"] = wrapperInfo["project"]
		info["GCP KMS Region"] = wrapperInfo["region"]
		info["GCP KMS Key Ring"] = wrapperInfo["key_ring"]
		info["GCP KMS Crypto Key"] = wrapperInfo["crypto_key"]
	}
	return wrapper, info, nil
}

var GetOCIKMSKMSFunc = func(opts *wrapping.WrapperOptions, kms *KMS) (wrapping.Wrapper, map[string]string, error) {
	wrapper := ocikms.NewWrapper(opts)
	wrapperInfo, err := wrapper.SetConfig(kms.Config)
	if err != nil {
		return nil, nil, err
	}
	info := make(map[string]string)
	if wrapperInfo != nil {
		info["OCI KMS KeyID"] = wrapperInfo[ocikms.KMSConfigKeyID]
		info["OCI KMS Crypto Endpoint"] = wrapperInfo[ocikms.KMSConfigCryptoEndpoint]
		info["OCI KMS Management Endpoint"] = wrapperInfo[ocikms.KMSConfigManagementEndpoint]
		info["OCI KMS Principal Type"] = wrapperInfo["principal_type"]
	}
	return wrapper, info, nil
}

var GetTransitKMSFunc = func(opts *wrapping.WrapperOptions, kms *KMS) (wrapping.Wrapper, map[string]string, error) {
	wrapper := transit.NewWrapper(opts)
	wrapperInfo, err := wrapper.SetConfig(kms.Config)
	if err != nil {
		// If the error is any other than logical.KeyNotFoundError, return the error
		if !errwrap.ContainsType(err, new(logical.KeyNotFoundError)) {
			return nil, nil, err
		}
	}
	info := make(map[string]string)
	if wrapperInfo != nil {
		info["Transit Address"] = wrapperInfo["address"]
		info["Transit Mount Path"] = wrapperInfo["mount_path"]
		info["Transit Key Name"] = wrapperInfo["key_name"]
		if namespace, ok := wrapperInfo["namespace"]; ok {
			info["Transit Namespace"] = namespace
		}
	}
	return wrapper, info, nil
}

func createSecureRandomReader(conf *SharedConfig, wrapper wrapping.Wrapper) (io.Reader, error) {
	return rand.Reader, nil
}
