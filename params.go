package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/exp/slices"
)

type paramsType struct {
	BotToken          string
	EasyDiffusionPath string

	AllowedUserIDs  []int64
	AdminUserIDs    []int64
	AllowedGroupIDs []int64

	DelayedEDStart bool
}

var params paramsType

func (p *paramsType) Init() error {
	flag.StringVar(&p.BotToken, "bot-token", "", "telegram bot token")
	flag.StringVar(&p.EasyDiffusionPath, "easy-diffusion-path", "", "path of the easy diffusion start script")
	var allowedUserIDs string
	flag.StringVar(&allowedUserIDs, "allowed-user-ids", "", "allowed telegram user ids")
	var adminUserIDs string
	flag.StringVar(&adminUserIDs, "admin-user-ids", "", "admin telegram user ids")
	var allowedGroupIDs string
	flag.StringVar(&allowedGroupIDs, "allowed-group-ids", "", "allowed telegram group ids")
	flag.BoolVar(&p.DelayedEDStart, "delayed-ed-start", false, "start easy diffusion only when the first prompt arrives")
	flag.Parse()

	if p.BotToken == "" {
		p.BotToken = os.Getenv("BOT_TOKEN")
	}
	if p.BotToken == "" {
		return fmt.Errorf("bot token not set")
	}

	if p.EasyDiffusionPath == "" {
		p.EasyDiffusionPath = os.Getenv("EASY_DIFFUSION_PATH")
	}
	if p.EasyDiffusionPath == "" {
		return fmt.Errorf("easy diffusion path not set")
	}

	if allowedUserIDs == "" {
		allowedUserIDs = os.Getenv("ALLOWED_USERIDS")
	}
	sa := strings.Split(allowedUserIDs, ",")
	for _, idStr := range sa {
		if idStr == "" {
			continue
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return fmt.Errorf("allowed user ids contains invalid user ID: " + idStr)
		}
		p.AllowedUserIDs = append(p.AllowedUserIDs, id)
	}

	if adminUserIDs == "" {
		adminUserIDs = os.Getenv("ADMIN_USERIDS")
	}
	sa = strings.Split(adminUserIDs, ",")
	for _, idStr := range sa {
		if idStr == "" {
			continue
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return fmt.Errorf("admin ids contains invalid user ID: " + idStr)
		}
		p.AdminUserIDs = append(p.AdminUserIDs, id)
		if !slices.Contains(p.AllowedUserIDs, id) {
			p.AllowedUserIDs = append(p.AllowedUserIDs, id)
		}
	}

	if allowedGroupIDs == "" {
		allowedGroupIDs = os.Getenv("ALLOWED_GROUPIDS")
	}
	sa = strings.Split(allowedGroupIDs, ",")
	for _, idStr := range sa {
		if idStr == "" {
			continue
		}
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return fmt.Errorf("allowed group ids contains invalid group ID: " + idStr)
		}
		p.AllowedGroupIDs = append(p.AllowedGroupIDs, id)
	}

	s := os.Getenv("DELAYED_ED_START")
	if s != "" {
		if s == "0" {
			p.DelayedEDStart = false
		} else {
			p.DelayedEDStart = true
		}
	}

	return nil
}
