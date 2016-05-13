package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/fsouza/go-dockerclient"
)

var dockerHooks DockerHooks

// Docker figures out what we should do with the hook and preforms the action
// on the docker deamon
type DockerHooks struct {
	lock   sync.Mutex
	client docker.Client
}

func NewDockerHooks() (*DockerHooks, error) {
	client, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}
	d := &DockerHooks{
		lock:   new(sync.Mutex),
		client: client,
	}
	return d, nil
}

// Act takes some action based on the hook
func (d *DockerHooks) Act(w Webhook) error {
	// endpoint := "unix:///var/run/docker.sock"
	imgs, _ := d.client.ListImages(docker.ListImagesOptions{All: false})
	for _, img := range imgs {
		fmt.Println("ID: ", img.ID)
		fmt.Println("RepoTags: ", img.RepoTags)
		fmt.Println("Created: ", img.Created)
		fmt.Println("Size: ", img.Size)
		fmt.Println("VirtualSize: ", img.VirtualSize)
		fmt.Println("ParentId: ", img.ParentID)
	}
	return nil
}

// Repository  contains the information sent in the repository field of the
// Webhook
type Repository struct {
	CommentCount string  `json:"comment_count"`
	DateCreated  float64 `json:"date_created"`
	Description  string  `json:"description"`
	Dockerfile   string  `json:"dockerfile"`
	IsOfficial   bool    `json:"is_official"`
	IsPrivate    bool    `json:"is_private"`
	IsTrusted    bool    `json:"is_trusted"`
	Name         string  `json:"name"`
	Namespace    string  `json:"namespace"`
	Owner        string  `json:"owner"`
	RepoName     string  `json:"repo_name"`
	RepoURL      string  `json:"repo_url"`
	StarCount    int     `json:"star_count"`
	Status       string  `json:"status"`
}

// Webhook contains the information sent by docker hub
type Webhook struct {
	CallbackURL string  `json:"callback_url"`
	PushedAt    float64 `json:"pushed_at"`
	Pusher      string  `json:"pusher"`
	Repository  Repository
}

// WebhookHandler receives Webhooks
func WebhookHandler(w http.ResponseWriter, r *http.Request) {
	// Decode the webhook
	decoder := json.NewDecoder(r.Body)
	var hook Webhook
	err := decoder.Decode(&hook)
	// If there was an erro get out of here
	if err != nil {
		log.Println("ERROR: parsing hook:", err)
		return
	}
	// If ther was no erro figure out what to do next
	log.Println(hook)
	// Act on the hook
	dockerHooks.Act(hook)
}

func main() {
	// Get the port from the env
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}
	// Initialize our hook responder
	dockerHooks, err = NewDockerHooks()
	if err != nil {
		log.Fatal(err)
	}
	// Register our hook handler
	http.HandleFunc("/", WebhookHandler)
	// Server at the requested port
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
