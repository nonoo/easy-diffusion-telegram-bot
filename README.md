# easy-diffusion-telegram-bot

This is a Telegram Bot frontend for rendering images with
[Easy Diffusion](https://github.com/easydiffusion/easydiffusion).

<p align="center"><img src="demo.gif?raw=true"/></p>

The bot displays the progress and further information during processing by
responding to the message with the prompt. Requests are queued, only one gets
processed at a time.

The bot uses the
[Telegram Bot API](https://github.com/go-telegram-bot-api/telegram-bot-api).
Rendered images are not saved on disk. Tested on Linux, but should be able
to run on other operating systems.

## Compiling

You'll need Go installed on your computer. Install a recent package of `golang`.
Then:

```
go get github.com/nonoo/easy-diffusion-telegram-bot
go install github.com/nonoo/easy-diffusion-telegram-bot
```

This will typically install `easy-diffusion-telegram-bot` into `$HOME/go/bin`.

Or just enter `go build` in the cloned Git source repo directory.

## Prerequisites

Create a Telegram bot using [BotFather](https://t.me/BotFather) and get the
bot's `token`.

## Running

You can get the available command line arguments with `-h`.
Mandatory arguments are:

- `-bot-token`: set this to your Telegram bot's `token`
- `-easy-diffusion-path`: set this to the path of `start.sh` from the Easy
  Diffusion directory

Set your Telegram user ID as an admin with the `-admin-user-ids` argument.
Admins will get a message when the bot starts.

Other user/group IDs can be set with the `-allowed-user-ids` and
`-allowed-group-ids` arguments. IDs should be separated by commas.

You can get Telegram user IDs by writing a message to the bot and checking
the app's log, as it logs all incoming messages.

All command line arguments can be set through OS environment variables.
Note that using a command line argument overwrites a setting by the environment
variable. Available OS environment variables are:

- `BOT_TOKEN`
- `EASY_DIFFUSION_PATH`
- `ALLOWED_USERIDS`
- `ADMIN_USERIDS`
- `ALLOWED_GROUPIDS`
- `DELAYED_ED_START`

## Supported commands

- `/ed` - Render images using supplied prompt
- `/edcancel` - Cancel ongoing download
- `/edhelp` - Cancel ongoing download

You can also use the `!` command character instead of `/`.

You don't need to enter the `/ed` command if you send a prompt to the bot using
a private chat.

### Setting render parameters

You can use the following `attr:val` assignments in the prompt:

- `seed/s` - set seed (hexadecimal)
- `width/w` - set output image width
- `height/h` - set output image height
- `infsteps/i` - set the number of inference steps
- `outcnt/o` - set count of output images
- `gscale/g` - set guidance scale
- `sampler/r` - set sampler, valid values are:
  - `plms`
  - `ddim`
  - `heun`
  - `euler`
  - `euler_a`
  - `dpm2`
  - `dpm2_a`
  - `lms`
  - `dpm_solver_stability`
  - `dpmpp_2s_a`
  - `dpmpp_2m`
  - `dpmpp_2m_sde`
  - `dpmpp_sde` (default)
  - `dpm_adaptive`
  - `ddpm`
  - `deis`
  - `unipc_snr`
  - `unipc_tu`
  - `unipc_snr_2`
  - `unipc_tu_2`
  - `unipc_tq`
- `model/m` - set model version:
  - 1: sd-v1-4
  - 2: [v1-5-pruned-emaonly](https://huggingface.co/runwayml/stable-diffusion-v1-5)
  - 3: [768-v-ema](https://huggingface.co/stabilityai/stable-diffusion-2)

Example prompt with attributes: `laughing santa with beer s:1 o:1`
Negative prompt example: `laughing santa with beer -:red_hat -:beer` (negative prompt will be translated to `red hat, beer`)

## Donations

If you find this bot useful then [buy me a beer](https://paypal.me/ha2non). :)
