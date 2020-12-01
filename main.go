package main

import (
	"context"
	"io"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

func main() {
	f, err := os.OpenFile("cache-daemon.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer f.Close()
	log.SetOutput(f)

	dockerClient, err := client.NewClientWithOpts(client.WithVersion("1.40"))
	if err != nil {
		log.Fatalf("Failed to build docker client: %v", err)
	}
	ctx := context.Background()
	for {
		images, err := imageList(ctx, dockerClient)
		if err != nil {
			log.Println("Error retrieving list of images: ", err)
		} else {
			for _, image := range images {
				if len(image.RepoTags) > 0 {
					log.Println("Retrieving image with tags " + image.RepoTags[0])
					imagePull(ctx, dockerClient, image.RepoTags[0])
				} else {
					log.Println("No repo tags found on image")
				}
			}
		}
		time.Sleep(60 * time.Second)
	}
}

func imageList(ctx context.Context, dockerClient *client.Client) ([]types.ImageSummary, error) {
	return dockerClient.ImageList(ctx, types.ImageListOptions{
		All:     true,
		Filters: filters.NewArgs(),
	})
}

func imagePull(ctx context.Context, dockerClient *client.Client, imageRef string) {
	reader, err := dockerClient.ImagePull(ctx, imageRef, types.ImagePullOptions{
		All:          false,
		RegistryAuth: "",
		PrivilegeFunc: func() (string, error) {
			// Can retry if authorization error occurs, this gives
			// registry authentication header value as base64 or an error
			// if this function will fail.
			// Doesn't matter to us.
			return "", nil
		},
		Platform: runtime.GOOS,
	})
	bytes := make([]byte, 1024)
	if err != nil {
		log.Println("Error:", err)
	} else {
		defer reader.Close()
		for {
			_, err := reader.Read(bytes)
			if err != nil {
				if err == io.EOF {
					break
				}
			}
			log.Println(string(bytes), imageRef)
		}
	}
}
