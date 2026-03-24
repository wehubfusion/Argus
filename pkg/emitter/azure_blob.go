package emitter

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"go.uber.org/zap"
)

// AzureBlobUploader implements BlobUploader for Azure Blob Storage using shared-key credentials.
// Use for monitoring lifecycle payloads when a dedicated storage client is needed.
type AzureBlobUploader struct {
	client        *azblob.Client
	serviceURL    string
	containerName string
	containerInit bool
	logger        *zap.Logger
}

// NewAzureBlobUploader creates an Azure Blob Storage uploader from a connection string.
// Supports HTTP endpoints (e.g. Azurite) for local development.
func NewAzureBlobUploader(connectionString, containerName string, logger *zap.Logger) (*AzureBlobUploader, error) {
	if connectionString == "" {
		return nil, fmt.Errorf("connection string is required")
	}
	if containerName == "" {
		return nil, fmt.Errorf("container name is required")
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	params := parseConnectionString(connectionString)
	accountName := params["AccountName"]
	accountKey := params["AccountKey"]
	serviceURL := params["BlobEndpoint"]
	if accountName == "" || accountKey == "" {
		return nil, fmt.Errorf("account name and key are required in the connection string")
	}
	if serviceURL == "" {
		serviceURL = fmt.Sprintf("https://%s.blob.core.windows.net", accountName)
	}

	credential, err := azblob.NewSharedKeyCredential(accountName, accountKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create shared key credential: %w", err)
	}

	var clientOpts *azblob.ClientOptions
	if strings.HasPrefix(strings.ToLower(serviceURL), "http://") {
		clientOpts = &azblob.ClientOptions{
			ClientOptions: azcore.ClientOptions{
				InsecureAllowCredentialWithHTTP: true,
			},
		}
	}

	client, err := azblob.NewClientWithSharedKeyCredential(serviceURL, credential, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create blob client: %w", err)
	}

	return &AzureBlobUploader{
		client:        client,
		serviceURL:    strings.TrimRight(serviceURL, "/"),
		containerName: containerName,
		logger:        logger,
	}, nil
}

// Upload implements BlobUploader. Uploads data to the configured container at the given path.
func (a *AzureBlobUploader) Upload(ctx context.Context, path string, data []byte, metadata map[string]string) (string, int64, error) {
	if err := a.ensureContainer(ctx); err != nil {
		return "", 0, err
	}

	metadataPtr := make(map[string]*string, len(metadata))
	for k, v := range metadata {
		metadataPtr[k] = to.Ptr(v)
	}

	containerClient := a.client.ServiceClient().NewContainerClient(a.containerName)
	blobClient := containerClient.NewBlockBlobClient(path)

	_, err := blobClient.UploadBuffer(ctx, data, &azblob.UploadBufferOptions{
		Metadata: metadataPtr,
		HTTPHeaders: &blob.HTTPHeaders{
			BlobContentType: to.Ptr("application/json"),
		},
	})
	if err != nil {
		a.logger.Warn("Failed to upload monitoring blob",
			zap.String("path", path),
			zap.Int("size", len(data)),
			zap.Error(err))
		return "", 0, fmt.Errorf("blob upload failed: %w", err)
	}

	return blobClient.URL(), int64(len(data)), nil
}

func (a *AzureBlobUploader) ensureContainer(ctx context.Context) error {
	if a.containerInit {
		return nil
	}

	_, err := a.client.CreateContainer(ctx, a.containerName, nil)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "containeralreadyexists") {
			a.containerInit = true
			return nil
		}
		var respErr *azcore.ResponseError
		if errors.As(err, &respErr) && respErr.ErrorCode == "ContainerAlreadyExists" {
			a.containerInit = true
			return nil
		}
		return fmt.Errorf("failed to ensure container: %w", err)
	}

	a.containerInit = true
	return nil
}

func parseConnectionString(connectionString string) map[string]string {
	parts := strings.Split(connectionString, ";")
	params := make(map[string]string, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.Index(part, "=")
		if idx <= 0 {
			continue
		}
		params[strings.TrimSpace(part[:idx])] = strings.TrimSpace(part[idx+1:])
	}
	return params
}
