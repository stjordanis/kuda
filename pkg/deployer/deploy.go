package deployer

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cyrildiagne/kuda/pkg/config"
	"github.com/cyrildiagne/kuda/pkg/utils"
)

func deployFromPublished(fromPublished string, env *Env, w http.ResponseWriter, r *http.Request) error {
	// Retrieve namespace.
	namespace, err := GetAuthorizedNamespace(env, r)
	if err != nil {
		return err
	}

	// Parse fromPublished to get author, name & version.
	im := ImageName{}
	if err := im.ParseFrom(fromPublished); err != nil {
		return err
	}

	// Check if image@version exists and is public.
	api, err := GetVersion(env, im)
	if err != nil {
		return err
	}
	if !api.IsPublic {
		err := fmt.Errorf("%s not found or not available", fromPublished)
		return StatusError{400, err}
	}

	// Generate Knative YAML with appropriate namespace.
	service := config.ServiceSummary{
		Name:           im.Name,
		Namespace:      namespace,
		DockerArtifact: env.GetDockerImagePath(im),
	}
	knativeCfg, err := config.GenerateKnativeConfig(service, api.Manifest.Deploy)
	if err != nil {
		return err
	}
	knativeYAML, err := config.MarshalKnativeConfig(knativeCfg)
	if err != nil {
		return err
	}
	// Create new temp directory.
	tempDir, err := ioutil.TempDir("", namespace+"__"+im.Name)
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)
	knativeFile := filepath.FromSlash(tempDir + "/knative.yaml")
	if err := utils.WriteYAML(knativeYAML, knativeFile); err != nil {
		return err
	}

	// Run kubectl apply.
	args := []string{"apply", "-f", knativeFile}
	if err := RunCMD(w, "kubectl", args); err != nil {
		return err
	}

	// TODO: Add to the namespaces' deployments.

	return nil
}

// HandleDeploy handles deployments from tar archived in body & published images.
func HandleDeploy(env *Env, w http.ResponseWriter, r *http.Request) error {
	// Set maximum upload size to 2GB.
	r.ParseMultipartForm((2 * 1000) << 20)

	// Retrieve namespace.
	namespace, err := GetAuthorizedNamespace(env, r)
	if err != nil {
		return err
	}

	// Check if deploying from published
	fromPublished := r.FormValue("from_published")
	if fromPublished != "" {
		return deployFromPublished(fromPublished, env, w, r)
	}

	// Extract archive to temp folder.
	contextDir, err := extractContext(namespace, r)
	if err != nil {
		return err
	}
	defer os.RemoveAll(contextDir) // Clean up.

	// Build and push image.
	if err := generate(namespace, contextDir, env); err != nil {
		return err
	}

	// Setup client stream.
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/event-stream")

	// // Build with Skaffold.
	if err := Skaffold("run", contextDir, contextDir+"/skaffold.yaml", w); err != nil {
		return err
	}

	// Load the manifest.
	manifestFile := filepath.FromSlash(contextDir + "/kuda.yaml")
	manifest, err := utils.LoadManifest(manifestFile)
	if err != nil {
		return StatusError{400, err}
	}

	// Register Template.
	apiVersion := APIVersion{
		IsPublic: false,
		Version:  manifest.Version,
		Manifest: manifest,
	}
	if err := registerAPI(env, namespace, apiVersion); err != nil {
		return err
	}

	// TODO: Add to the namespaces' deployments.

	fmt.Fprintf(w, "Deployment successful!\n")
	return nil
}
