package main

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/gorilla/mux"
)

type Response struct {
	Status  string      `json:"status"`
	Data    interface{} `json:"data"`
	Success bool        `json:"success"`
}

type ContainerResponse struct {
	ID string `json:"id"`
}

func createTarContext(dockerfilePath string) (*bytes.Buffer, error) {
	tarBuffer := new(bytes.Buffer)
	tarWriter := tar.NewWriter(tarBuffer)
	defer tarWriter.Close()

	file, err := os.Open(dockerfilePath)
	if err != nil {
		return nil, fmt.Errorf("erro ao abrir Dockerfile: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("erro ao obter informações do Dockerfile: %w", err)
	}

	header := &tar.Header{
		Name: "Dockerfile",
		Size: stat.Size(),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		return nil, fmt.Errorf("erro ao escrever cabeçalho TAR: %w", err)
	}

	if _, err := io.Copy(tarWriter, file); err != nil {
		return nil, fmt.Errorf("erro ao copiar conteúdo do Dockerfile para TAR: %w", err)
	}

	return tarBuffer, nil
}

func HandleStartDockerContainer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	ctx := context.Background()

	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao criar client Docker: %v", err), http.StatusInternalServerError)
		return
	}

	dockerfilePath := "./Dockerfile"

	tarContext, err := createTarContext(dockerfilePath)
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao criar contexto TAR: %v", err), http.StatusInternalServerError)
		return
	}

	imageBuildResponse, err := cli.ImageBuild(ctx, tarContext, types.ImageBuildOptions{
		Tags: []string{"local-dockerfile-image:latest"},
	})

	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao construir imagem: %v", err), http.StatusInternalServerError)
		return
	}

	if _, err := io.Copy(os.Stdout, imageBuildResponse.Body); err != nil {
		http.Error(w, fmt.Sprintf("Erro ao ler a saída da construção da imagem: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Println("Imagem Docker construída com sucesso.")

	containerConfig := &container.Config{
		Image: "local-dockerfile-image:latest",
		Tty:   true,
	}

	resp, err := cli.ContainerCreate(ctx, containerConfig, nil, nil, nil, "container-from-dockerfile")
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao criar container: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Printf("Container criado com ID: %s\n", resp.ID)

	// Inicia o container
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		http.Error(w, fmt.Sprintf("Erro ao iniciar container: %v", err), http.StatusInternalServerError)
		return
	}

	response := Response{
		Data: []ContainerResponse{
			{
				ID: resp.ID,
			},
		},
		Status:  "success",
		Success: true,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Erro ao gerar JSON", http.StatusInternalServerError)
	}
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/create-container", HandleStartDockerContainer).Methods("GET")

	log.Println("Servidor iniciado na porta 8080")
	http.ListenAndServe(":8080", r)
}
