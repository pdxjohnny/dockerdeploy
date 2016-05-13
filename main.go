package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/fsouza/go-dockerclient"
)

const (
	RestartTimeout = 10
)

var dockerHooks *DockerHooks

// Docker figures out what we should do with the hook and preforms the action
// on the docker deamon
type DockerHooks struct {
	lock   sync.Mutex
	client *docker.Client
}

func NewDockerHooks() (*DockerHooks, error) {
	// endpoint := "unix:///var/run/docker.sock"
	client, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}
	d := &DockerHooks{
		client: client,
	}
	return d, nil
}

// Act takes some action based on the hook
func (d *DockerHooks) Act(w Webhook) error {
	log.Println("Got hook for", w.Repository.RepoName)
	conts, err := d.client.ListContainers(docker.ListContainersOptions{
		All: true,
		Filters: map[string][]string{
			"status": []string{"running"},
		},
	})
	if err != nil {
		return err
	}
	restart := make([]string, 0)
	for _, cont := range conts {
		if cont.Image == w.Repository.RepoName {
			restart = append(restart, cont.ID)
		}
	}
	// Nothing to restart
	if len(restart) < 1 {
		log.Println("Nothing to update for", w.Repository.RepoName)
		return nil
	}
	// We have something to restart so pull the latest image
	d.client.PullImage(
		docker.PullImageOptions{
			Repository:   w.Repository.RepoName,
			OutputStream: os.Stdout,
		},
		docker.AuthConfiguration{},
	)
	// Now restart all of the conatiners using that image
	for _, id := range restart {
		log.Println("Restarting", w.Repository.RepoName, id)
		d.client.RestartContainer(id, RestartTimeout)
	}
	return nil
}

// Repository  contains the information sent in the repository field of the
// Webhook
type Repository struct {
	CommentCount    string  `json:"comment_count"`
	DateCreated     float64 `json:"date_created"`
	Description     string  `json:"description"`
	Dockerfile      string  `json:"dockerfile"`
	FullDescription string  `json:"full_description"`
	IsOfficial      bool    `json:"is_official"`
	IsPrivate       bool    `json:"is_private"`
	IsTrusted       bool    `json:"is_trusted"`
	Name            string  `json:"name"`
	Namespace       string  `json:"namespace"`
	Owner           string  `json:"owner"`
	RepoName        string  `json:"repo_name"`
	RepoURL         string  `json:"repo_url"`
	StarCount       int     `json:"star_count"`
	Status          string  `json:"status"`
}

// Webhook contains the information sent by docker hub
type Webhook struct {
	CallbackURL string          `json:"callback_url"`
	PushData    json.RawMessage `json:"push_data"`
	PushedAt    float64         `json:"pushed_at"`
	Pusher      string          `json:"pusher"`
	Repository  Repository
}

// WebhookCallback is how we tell docker hub that all went well
type WebhookCallback struct {
	State       string `json:"state"`
	Description string `json:"description"`
	Context     string `json:"context"`
	TargetURL   string `json:"target_url"`
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
	// If ther was no err figure out what to do next
	dockerHooks.Act(hook)
}

func main() {
	var err error
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
