package main

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const processStartStr = "🛎 Starting render..."
const processStr = "🔨 Processing"
const progressBarLength = 20
const uploadingStr = "☁ ️ Uploading..."
const errorStr = "❌ Error"
const canceledStr = "❌ Canceled"

const processTimeout = 3 * time.Minute

type DownloadQueueEntry struct {
	Params RenderParams

	TaskID           uint64
	RenderParamsText string

	ReplyMessage *models.Message
	Message      *models.Message
}

func (e *DownloadQueueEntry) sendReply(ctx context.Context, s string) {
	if e.ReplyMessage == nil {
		e.ReplyMessage = sendReplyToMessage(ctx, e.Message, s)
	} else {
		_, _ = telegramBot.EditMessageText(ctx, &bot.EditMessageTextParams{
			MessageID: e.ReplyMessage.ID,
			ChatID:    e.ReplyMessage.Chat.ID,
			Text:      s,
		})
	}
}

func (e *DownloadQueueEntry) sendImages(ctx context.Context, imgs [][]byte) {
	if len(imgs) == 0 {
		return
	}

	var media []models.InputMedia
	for i := range imgs {
		var c string
		if i == 0 {
			c = e.Params.OrigPrompt + " (" + e.RenderParamsText + ")"
		}
		media = append(media, &models.InputMediaPhoto{
			Media:           fmt.Sprintf("attach://ed-image-%x-%d-%d.jpg", e.Params.Seed, e.TaskID, i),
			MediaAttachment: bytes.NewReader(imgs[i]),
			Caption:         c,
		})
	}
	params := &bot.SendMediaGroupParams{
		ChatID:           e.Message.Chat.ID,
		ReplyToMessageID: e.Message.ID,
		Media:            media,
	}
	_, _ = telegramBot.SendMediaGroup(ctx, params)
}

func (e *DownloadQueueEntry) deleteReply(ctx context.Context) {
	if e.ReplyMessage == nil {
		return
	}

	_, _ = telegramBot.DeleteMessage(ctx, &bot.DeleteMessageParams{
		MessageID: e.ReplyMessage.ID,
		ChatID:    e.ReplyMessage.Chat.ID,
	})
}

type DownloadQueueCurrentEntry struct {
	canceled  bool
	ctxCancel context.CancelFunc
}

type DownloadQueue struct {
	mutex          sync.Mutex
	ctx            context.Context
	entries        []DownloadQueueEntry
	processReqChan chan bool

	currentEntry DownloadQueueCurrentEntry
}

func (q *DownloadQueue) Add(params RenderParams, message *models.Message) {
	q.mutex.Lock()

	var replyStr string
	if len(q.entries) == 0 {
		replyStr = processStartStr
	} else {
		fmt.Println("  queueing request at position #", len(q.entries))
		replyStr = q.getQueuePositionString(len(q.entries))
	}

	newEntry := DownloadQueueEntry{
		Params:  params,
		Message: message,
	}

	newEntry.sendReply(q.ctx, replyStr)

	q.entries = append(q.entries, newEntry)
	q.mutex.Unlock()

	select {
	case q.processReqChan <- true:
	default:
	}
}

func (q *DownloadQueue) CancelCurrentEntry(ctx context.Context) (err error) {
	q.mutex.Lock()
	if len(q.entries) > 0 {
		q.currentEntry.canceled = true
		q.currentEntry.ctxCancel()
	} else {
		fmt.Println("  no active request to cancel")
		err = fmt.Errorf("no active request to cancel")
	}
	q.mutex.Unlock()
	return
}

func (q *DownloadQueue) getQueuePositionString(pos int) string {
	return "👨‍👦‍👦 Request queued at position #" + fmt.Sprint(pos)
}

func (q *DownloadQueue) sendProgressReply(qEntry *DownloadQueueEntry, progress int) {
	qEntry.sendReply(q.ctx, processStr+" "+getProgressbar(progress, progressBarLength)+"\n"+qEntry.RenderParamsText)
}

func (q *DownloadQueue) queryProgress(qEntry *DownloadQueueEntry, prevProgress int) (progress int, imgs [][]byte, err error) {
	progress = prevProgress

	var newProgress int
	newProgress, imgs, err = req.GetProgress(qEntry.TaskID)
	if err == nil && newProgress > prevProgress {
		progress = newProgress
		fmt.Print("    progress: ", progress, "%\n")
	}
	return
}

func (q *DownloadQueue) processQueueEntry(renderCtx context.Context, qEntry *DownloadQueueEntry) error {
	fmt.Print("processing request from ", qEntry.Message.From.Username, "#", qEntry.Message.From.ID, ": ", qEntry.Params.Prompt, "\n")

	qEntry.RenderParamsText = fmt.Sprintf("🌱0x%X 👟%d 🕹%.1f 🖼%dx%dx%d 🔭%s", qEntry.Params.Seed, qEntry.Params.NumInferenceSteps,
		qEntry.Params.GuidanceScale, qEntry.Params.Width, qEntry.Params.Height, qEntry.Params.NumOutputs, qEntry.Params.SamplerName)

	qEntry.sendReply(q.ctx, processStartStr+"\n"+qEntry.RenderParamsText)

	var err error
	qEntry.TaskID, err = req.Render(qEntry.Params)
	if err != nil {
		return err
	}
	fmt.Println("  render started with task id", qEntry.TaskID)

	progress, imgs, err := q.queryProgress(qEntry, 0)
	if err != nil {
		return err
	}
	if imgs == nil {
		q.sendProgressReply(qEntry, progress)

		progressPercentUpdateTicker := time.NewTicker(500 * time.Millisecond)
		progressCheckTicker := time.NewTicker(100 * time.Millisecond)
	checkLoop:
		for {
			select {
			case <-renderCtx.Done():
				return fmt.Errorf("timeout")
			case <-progressPercentUpdateTicker.C:
				q.sendProgressReply(qEntry, progress)
			case <-progressCheckTicker.C:
				progress, imgs, err = q.queryProgress(qEntry, progress)
				if err != nil {
					return err
				}
				if imgs != nil {
					break checkLoop
				}
			}
		}
	}

	qEntry.sendReply(q.ctx, uploadingStr+"\n"+qEntry.RenderParamsText)
	qEntry.sendImages(q.ctx, imgs)
	qEntry.deleteReply(q.ctx)

	return nil
}

func (q *DownloadQueue) processor() {
	for {
		q.mutex.Lock()
		if (len(q.entries)) == 0 {
			q.mutex.Unlock()
			<-q.processReqChan
			continue
		}

		// Updating queue positions for all waiting entries.
		for i := 1; i < len(q.entries); i++ {
			sendReplyToMessage(q.ctx, q.entries[i].Message, q.getQueuePositionString(i))
		}

		qEntry := &q.entries[0]

		q.currentEntry = DownloadQueueCurrentEntry{}
		var renderCtx context.Context
		renderCtx, q.currentEntry.ctxCancel = context.WithTimeout(q.ctx, processTimeout)
		q.mutex.Unlock()

		err := q.processQueueEntry(renderCtx, qEntry)

		q.mutex.Lock()
		if q.currentEntry.canceled {
			fmt.Print("  canceled\n")
			req.Stop(qEntry.TaskID)
			qEntry.sendReply(q.ctx, canceledStr)
		} else if err != nil {
			fmt.Println("  error:", err)
			qEntry.sendReply(q.ctx, errorStr+": "+err.Error())
		}

		q.currentEntry.ctxCancel()

		q.entries = q.entries[1:]
		if len(q.entries) == 0 {
			fmt.Print("finished queue processing\n")
		}
		q.mutex.Unlock()
	}
}

func (q *DownloadQueue) Init(ctx context.Context) {
	q.ctx = ctx
	q.processReqChan = make(chan bool)
	go q.processor()
}
