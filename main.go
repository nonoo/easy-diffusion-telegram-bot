package main

import (
	"context"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"golang.org/x/exp/slices"
)

var telegramBot *bot.Bot
var req ReqType
var dlQueue DownloadQueue

func sendReplyToMessage(ctx context.Context, replyToMsg *models.Message, s string) (msg *models.Message) {
	var err error
	msg, err = telegramBot.SendMessage(ctx, &bot.SendMessageParams{
		ReplyToMessageID: replyToMsg.ID,
		ChatID:           replyToMsg.Chat.ID,
		Text:             s,
	})
	if err != nil {
		fmt.Println("  reply send error:", err)
	}
	return
}

func handleCmdED(ctx context.Context, msg *models.Message) {
	renderParams := RenderParams{
		OrigPrompt:        msg.Text,
		Seed:              rand.Uint32(),
		Width:             512,
		Height:            512,
		NumInferenceSteps: 20,
		NumOutputs:        4,
		GuidanceScale:     7,
		SamplerName:       params.DefaultSampler,
		ModelName:         params.DefaultModel,
	}

	var prompt []string
	var promptLine string

	lines := strings.Split(msg.Text, "\n")
	if len(lines) >= 2 {
		promptLine = strings.TrimSpace(lines[0])
		renderParams.NegativePrompt = strings.TrimSpace(strings.Join(lines[1:], " "))
	} else {
		promptLine = strings.TrimSpace(msg.Text)
	}

	words := strings.Split(promptLine, " ")
	for i := range words {
		words[i] = strings.TrimSpace(words[i])

		if words[i][0] != '-' { // Only process words starting with -
			prompt = append(prompt, words[i])
			continue
		}

		splitword := strings.Split(words[i], ":")
		if len(splitword) == 2 {
			attr := strings.ToLower(splitword[0][1:])
			val := splitword[1]

			switch attr {
			case "seed", "s":
				val = strings.TrimPrefix(val, "🌱")
				val = strings.TrimPrefix(val, "0x")
				valInt := new(big.Int)
				if _, ok := valInt.SetString(val, 16); !ok {
					fmt.Println("  invalid seed")
					sendReplyToMessage(ctx, msg, errorStr+": invalid seed")
					return
				}
				renderParams.Seed = uint32(valInt.Uint64())
			case "width", "w":
				valInt, err := strconv.Atoi(val)
				if err != nil {
					fmt.Println("  invalid width")
					sendReplyToMessage(ctx, msg, errorStr+": invalid width")
					return
				}
				renderParams.Width = valInt
			case "height", "h":
				valInt, err := strconv.Atoi(val)
				if err != nil {
					fmt.Println("  invalid height")
					sendReplyToMessage(ctx, msg, errorStr+": invalid height")
					return
				}
				renderParams.Height = valInt
			case "infsteps", "i":
				valInt, err := strconv.Atoi(val)
				if err != nil {
					fmt.Println("  invalid inference steps")
					sendReplyToMessage(ctx, msg, errorStr+": invalid inference steps")
					return
				}
				renderParams.NumInferenceSteps = valInt
			case "outcnt", "o":
				valInt, err := strconv.Atoi(val)
				if err != nil {
					fmt.Println("  invalid output count")
					sendReplyToMessage(ctx, msg, errorStr+": invalid output count")
					return
				}
				renderParams.NumOutputs = valInt
			case "gscale", "g":
				valFloat, err := strconv.ParseFloat(val, 32)
				if err != nil {
					fmt.Println("  invalid guidance scale")
					sendReplyToMessage(ctx, msg, errorStr+": invalid guidance scale")
					return
				}
				renderParams.GuidanceScale = float32(valFloat)
			case "sampler", "r":
				val = strings.ToLower(val)
				switch val {
				case "plms", "ddim", "heun", "euler", "euler_a", "dpm2", "dpm2_a", "lms",
					"dpm_solver_stability", "dpmpp_2s_a", "dpmpp_2m", "dpmpp_2m_sde",
					"dpmpp_sde", "dpm_adaptive", "ddpm", "deis", "unipc_snr", "unipc_tu",
					"unipc_snr_2", "unipc_tu_2", "unipc_tq":
					renderParams.SamplerName = val
				default:
					fmt.Println("  invalid sampler")
					sendReplyToMessage(ctx, msg, errorStr+": invalid sampler")
					return
				}
			case "model", "m":
				renderParams.ModelName = val
			default:
				fmt.Println("  invalid attribute", attr)
				sendReplyToMessage(ctx, msg, errorStr+": invalid attribute "+attr)
				return
			}
		}
	}

	renderParams.Prompt = strings.Join(prompt, " ")

	if renderParams.Prompt == "" {
		fmt.Println("  missing prompt")
		sendReplyToMessage(ctx, msg, errorStr+": missing prompt")
		return
	}

	dlQueue.Add(renderParams, msg)
}

func handleCmdEDCancel(ctx context.Context, msg *models.Message) {
	if err := dlQueue.CancelCurrentEntry(ctx); err != nil {
		sendReplyToMessage(ctx, msg, errorStr+": "+err.Error())
	}
}

func handleCmdModels(ctx context.Context, msg *models.Message) {
	modelsDir := filepath.Join(filepath.Dir(params.EasyDiffusionPath), "models", "stable-diffusion")
	files, err := os.ReadDir(modelsDir)
	if err != nil {
		fmt.Println("  can't list models directory:", err)
		sendReplyToMessage(ctx, msg, errorStr+": can't list models directory: "+err.Error())
		return
	}
	var models []string
	for _, file := range files {
		fn := file.Name()
		ext := filepath.Ext(fn)
		switch ext {
		case ".safetensors", ".ckpt":
			models = append(models, strings.TrimSuffix(fn, ext))
		}
	}
	sendReplyToMessage(ctx, msg, "🧩 Available models: "+strings.Join(models, ", ")+". Default: "+params.DefaultModel)
}

func handleCmdEmbeddings(ctx context.Context, msg *models.Message) {
	modelsDir := filepath.Join(filepath.Dir(params.EasyDiffusionPath), "models", "embeddings")
	files, err := os.ReadDir(modelsDir)
	if err != nil {
		fmt.Println("  can't list embeddings directory:", err)
		sendReplyToMessage(ctx, msg, errorStr+": can't list embeddings directory: "+err.Error())
		return
	}
	var embeddings []string
	for _, file := range files {
		fn := file.Name()
		ext := filepath.Ext(fn)
		switch ext {
		case ".pt":
			embeddings = append(embeddings, strings.TrimSuffix(fn, ext))
		}
	}
	sendReplyToMessage(ctx, msg, "Available embeddings: "+strings.Join(embeddings, ", "))
}

func handleCmdHelp(ctx context.Context, msg *models.Message) {
	sendReplyToMessage(ctx, msg, "🤖 Easy Diffusion Telegram Bot\n\n"+
		"Available commands:\n\n"+
		"!ed [prompt] - render prompt\n"+
		"!edcancel - cancel current render\n"+
		"!edmodels - list available models\n"+
		"!edembeddings - list available embeddings\n"+
		"!edhelp - show this help\n\n"+
		"For more information see https://github.com/nonoo/easy-diffusion-telegram-bot")
}

func telegramBotUpdateHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message == nil || update.Message.Text == "" { // Only handling message updates.
		return
	}

	fmt.Print("msg from ", update.Message.From.Username, "#", update.Message.From.ID, ": ", update.Message.Text, "\n")

	if update.Message.Chat.ID >= 0 { // From user?
		if !slices.Contains(params.AllowedUserIDs, update.Message.From.ID) {
			fmt.Println("  user not allowed, ignoring")
			return
		}
	} else { // From group ?
		fmt.Print("  msg from group #", update.Message.Chat.ID)
		if !slices.Contains(params.AllowedGroupIDs, update.Message.Chat.ID) {
			fmt.Println(", group not allowed, ignoring")
			return
		}
		fmt.Println()
	}

	// Check if message is a command.
	if update.Message.Text[0] == '/' || update.Message.Text[0] == '!' {
		cmd := strings.Split(update.Message.Text, " ")[0]
		if strings.Contains(cmd, "@") {
			cmd = strings.Split(cmd, "@")[0]
		}
		update.Message.Text = strings.TrimPrefix(update.Message.Text, cmd+" ")
		cmd = cmd[1:] // Cutting the command character.
		switch cmd {
		case "ed":
			handleCmdED(ctx, update.Message)
			return
		case "edcancel":
			handleCmdEDCancel(ctx, update.Message)
			return
		case "edmodels":
			handleCmdModels(ctx, update.Message)
			return
		case "edembeddings":
			handleCmdEmbeddings(ctx, update.Message)
			return
		case "edhelp":
			handleCmdHelp(ctx, update.Message)
			return
		case "start":
			fmt.Println("  (start cmd)")
			if update.Message.Chat.ID >= 0 { // From user?
				sendReplyToMessage(ctx, update.Message, "🤖 Welcome! This is a Telegram Bot frontend "+
					"for rendering images with Easy Diffusion.\n\nMore info: https://github.com/nonoo/easy-diffusion-telegram-bot")
			}
			return
		default:
			fmt.Println("  (invalid cmd)")
			if update.Message.Chat.ID >= 0 {
				sendReplyToMessage(ctx, update.Message, errorStr+": invalid command")
			}
			return
		}
	}

	if update.Message.Chat.ID >= 0 { // From user?
		handleCmdED(ctx, update.Message)
	}
}

func main() {
	fmt.Println("easy-diffusion-telegram-bot starting...")

	if err := params.Init(); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	if !params.DelayedEDStart {
		if err := startEasyDiffusionIfNeeded(); err != nil {
			panic(err.Error())
		}
	}

	var cancel context.CancelFunc
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	dlQueue.Init(ctx)

	opts := []bot.Option{
		bot.WithDefaultHandler(telegramBotUpdateHandler),
	}

	var err error
	telegramBot, err = bot.New(params.BotToken, opts...)
	if nil != err {
		panic(fmt.Sprint("can't init telegram bot: ", err))
	}

	for _, chatID := range params.AdminUserIDs {
		_, _ = telegramBot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: chatID,
			Text:   "🤖 Bot started",
		})
	}

	telegramBot.Start(ctx)
}
