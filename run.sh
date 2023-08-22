#!/bin/bash

. config.inc.sh

bin=./easy-diffusion-telegram-bot
if [ ! -x "$bin" ]; then
	bin="go run *.go"
fi

BOT_TOKEN=$BOT_TOKEN \
EASY_DIFFUSION_PATH=$EASY_DIFFUSION_PATH \
ALLOWED_USERIDS=$ALLOWED_USERIDS \
ADMIN_USERIDS=$ADMIN_USERIDS \
ALLOWED_GROUPIDS=$ALLOWED_GROUPIDS \
DELAYED_ED_START=$DELAYED_ED_START \
$bin $*
