package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const reqURL = "http://localhost:9000"

type ReqType struct{}

func (r *ReqType) req(path string, postData []byte) (string, error) {
	client := http.Client{
		Timeout: 3 * time.Second,
	}
	path, err := url.JoinPath(reqURL, path)
	if err != nil {
		return "", err
	}

	var request *http.Request
	if postData != nil {
		request, err = http.NewRequest("POST", path, bytes.NewBuffer(postData))
		if err != nil {
			return "", err
		}
		request.Header.Set("Content-Type", "application/json; charset=UTF-8")
	} else {
		request, err = http.NewRequest("GET", path, nil)
		if err != nil {
			return "", err
		}
	}

	resp, err := client.Do(request)
	if err != nil || resp.StatusCode != 200 {
		return "", err
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return "", err
	}
	return string(bodyBytes), nil
}

func (r *ReqType) Ping() (bool, error) {
	res, err := r.req("/ping", nil)
	if err != nil {
		return false, err
	}
	var pingRes struct {
		Status string `json:"status"`
	}
	err = json.Unmarshal([]byte(res), &pingRes)
	if err != nil {
		return false, err
	}
	return pingRes.Status == "Online", nil
}

type RenderReq struct {
	ActiveTags              []string `json:"active_tags"`
	BlockNSFW               bool     `json:"block_nsfw"`
	ClipSkip                bool     `json:"clip_skip"`
	GuidanceScale           float32  `json:"guidance_scale"`
	Height                  uint32   `json:"height"`
	InactiveTags            []string `json:"inactive_tags"`
	MetadataOutputFormat    string   `json:"metadata_output_format"`
	NegativePrompt          string   `json:"negative_prompt"`
	NumInferenceSteps       uint32   `json:"num_inference_steps"`
	NumOutputs              uint32   `json:"num_outputs"`
	OriginalPrompt          string   `json:"original_prompt"`
	OutputFormat            string   `json:"output_format"`
	OutputLossless          bool     `json:"output_lossless"`
	OutputQuality           uint32   `json:"output_quality"`
	Prompt                  string   `json:"prompt"`
	SamplerName             string   `json:"sampler_name"`
	Seed                    uint32   `json:"seed"`
	SessionID               string   `json:"session_id"`
	ShowOnlyFilteredImage   bool     `json:"show_only_filtered_image"`
	StreamImageProgress     bool     `json:"stream_image_progress"`
	StreamProgressUpdates   bool     `json:"stream_progress_updates"`
	Tiling                  string   `json:"tiling"`
	UseStableDiffusionModel string   `json:"use_stable_diffusion_model"`
	UseVaeModel             string   `json:"use_vae_model"`
	UsedRandomSeed          bool     `json:"used_random_seed"`
	VRAMUsageLevel          string   `json:"vram_usage_level"`
	Width                   uint32   `json:"width"`
}

type RenderParams struct {
	Prompt            string
	OrigPrompt        string
	Seed              uint32
	Width             int
	Height            int
	NumInferenceSteps int
	NumOutputs        int
	GuidanceScale     float32
	SamplerName       string
	ModelVersion      int
}

func (r *ReqType) Render(params RenderParams) (taskID uint64, err error) {
	var model string
	switch params.ModelVersion {
	default:
		model = "sd-v1-4"
	case 2:
		model = "v1-5-pruned-emaonly"
	case 3:
		model = "768-v-ema"
	}

	postData, err := json.Marshal(RenderReq{
		GuidanceScale:           params.GuidanceScale,
		Height:                  uint32(params.Height),
		MetadataOutputFormat:    "none",
		NumInferenceSteps:       uint32(params.NumInferenceSteps),
		NumOutputs:              uint32(params.NumOutputs),
		OriginalPrompt:          params.Prompt,
		OutputFormat:            "jpeg",
		OutputQuality:           75,
		Prompt:                  params.Prompt,
		SamplerName:             params.SamplerName,
		Seed:                    params.Seed,
		SessionID:               fmt.Sprint(rand.Uint32()),
		ShowOnlyFilteredImage:   true,
		StreamProgressUpdates:   true,
		Tiling:                  "none",
		UseStableDiffusionModel: model,
		UsedRandomSeed:          true,
		VRAMUsageLevel:          "high",
		Width:                   uint32(params.Width),
	})
	if err != nil {
		return 0, err
	}

	res, err := r.req("/render", postData)
	if err != nil {
		return 0, err
	}

	var renderResp struct {
		Status string `json:"status"`
		Task   uint64 `json:"task"`
	}
	err = json.Unmarshal([]byte(res), &renderResp)
	if err != nil || renderResp.Status != "Online" {
		return 0, err
	}
	if renderResp.Task == 0 {
		return 0, fmt.Errorf("unknown error")
	}

	return renderResp.Task, nil
}

func (r *ReqType) Stop(taskID uint64) {
	_, _ = r.req(fmt.Sprint("/image/stop?task=", taskID), nil)
}

func (r *ReqType) processProgressSection(section string) (progress int, imgs [][]byte, err error) {
	// Try to parse progress.
	var progressResp struct {
		Step       int `json:"step"`
		TotalSteps int `json:"total_steps"`
	}
	if marshalErr := json.Unmarshal([]byte(section), &progressResp); marshalErr == nil {
		if progressResp.TotalSteps > 0 {
			progress = int(float32(progressResp.Step*100) / float32(progressResp.TotalSteps))
		}
	}

	// Try to parse results.
	type resultRespOutput struct {
		Data string `json:"data"`
	}
	var resultResp struct {
		Status string             `json:"status"`
		Detail string             `json:"detail"`
		Output []resultRespOutput `json:"output"`
	}
	if marshalErr := json.Unmarshal([]byte(section), &resultResp); marshalErr == nil {
		if resultResp.Status != "" {
			if resultResp.Status == "succeeded" {
				progress = 100
				if len(resultResp.Output) == 0 {
					return progress, nil, fmt.Errorf("no images in result")
				}

				for _, output := range resultResp.Output {
					// Removing MIME type.
					var ok bool
					_, output.Data, ok = strings.Cut(output.Data, ",")
					if !ok {
						return progress, nil, fmt.Errorf("image base64 decode error")
					}

					var unbased []byte
					if unbased, err = base64.StdEncoding.DecodeString(output.Data); err != nil {
						return progress, nil, fmt.Errorf("image base64 decode error")
					}
					imgs = append(imgs, unbased)
				}
			} else {
				return progress, nil, fmt.Errorf("got status %s: %s", resultResp.Status, resultResp.Detail)
			}
		}
	}

	return progress, imgs, nil
}

func (r *ReqType) GetProgress(taskID uint64) (progress int, imgs [][]byte, err error) {
	var res string
	res, err = r.req(fmt.Sprint("/image/stream/", taskID), nil)
	if err != nil {
		return 0, nil, err
	}

	partStart := 0
	bracketCount := 0
	for i := 0; i < len(res); i++ {
		switch res[i] {
		case '{':
			bracketCount++
			if bracketCount == 1 {
				partStart = i
			}
		case '}':
			if bracketCount > 0 {
				bracketCount--
			}
			if bracketCount == 0 && partStart != i {
				progress, imgs, err = r.processProgressSection(res[partStart : i+1])
				if imgs != nil || err != nil {
					return progress, imgs, err
				}
			}
		}
	}

	return progress, imgs, nil
}
