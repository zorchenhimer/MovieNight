package main

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/zorchenhimer/MovieNight/common"
)

var colorRegex *regexp.Regexp = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

type CommandControl struct {
	user  map[string]Command
	mod   map[string]Command
	admin map[string]Command
}

type Command struct {
	HelpText string
	Function CommandFunction
}

type CommandFunction func(client *Client, args []string) string

//type HelpFunction func(client *Client) string

var commands = &CommandControl{
	user: map[string]Command{
		common.CNMe.String(): Command{
			HelpText: "Display an action message.",
			Function: func(client *Client, args []string) string {
				client.Me(strings.Join(args, " "))
				return ""
			},
		},

		common.CNHelp.String(): Command{
			HelpText: "This help text.",
			Function: cmdHelp,
		},

		common.CNCount.String(): Command{
			HelpText: "Display number of users in chat.",
			Function: func(client *Client, args []string) string {
				return fmt.Sprintf("Users in chat: %d", client.belongsTo.UserCount())
			},
		},

		common.CNColor.String(): cmdColor,

		common.CNWhoAmI.String(): cmdWhoAmI,

		common.CNAuth.String(): Command{
			HelpText: "Authenticate to admin",
			Function: func(cl *Client, args []string) string {
				if cl.IsAdmin {
					return "You are already authenticated."
				}

				pw := html.UnescapeString(strings.Join(args, " "))

				if settings.AdminPassword == pw {
					cl.IsMod = true
					cl.IsAdmin = true
					fmt.Printf("[auth] %s used the admin password\n", cl.name)
					return "Admin rights granted."
				}

				if cl.belongsTo.redeemModPass(pw) {
					cl.IsMod = true
					fmt.Printf("[auth] %s used a mod password\n", cl.name)
					return "Moderator privileges granted."
				}

				fmt.Printf("[auth] %s gave an invalid password\n", cl.name)
				return "Invalid password."
			},
		},

		common.CNUsers.String(): Command{
			HelpText: "Show a list of users in chat",
			Function: func(cl *Client, args []string) string {
				names := cl.belongsTo.GetNames()
				return strings.Join(names, " ")
			},
		},
	},

	mod: map[string]Command{
		common.CNSv.String(): Command{
			HelpText: "Send a server announcement message.  It will show up red with a border in chat.",
			Function: func(cl *Client, args []string) string {
				if len(args) == 0 {
					return "Missing message"
				}
				svmsg := formatLinks(strings.Join(ParseEmotesArray(args), " "))
				cl.belongsTo.AddMsg(cl, false, true, svmsg)
				return ""
			},
		},

		common.CNPlaying.String(): Command{
			HelpText: "Set the title text and info link.",
			Function: func(cl *Client, args []string) string {
				// Clear/hide title if sent with no arguments.
				if len(args) == 0 {
					cl.belongsTo.ClearPlaying()
					return ""
				}
				link := ""
				title := ""

				// pickout the link (can be anywhere, as long as there are no spaces).
				for _, word := range args {
					word = html.UnescapeString(word)
					if strings.HasPrefix(word, "http://") || strings.HasPrefix(word, "https://") {
						link = word
					} else {
						title = title + " " + word
					}
				}

				cl.belongsTo.SetPlaying(title, link)
				return ""
			},
		},

		common.CNUnmod.String(): Command{
			HelpText: "Revoke a user's moderator privilages.  Moderators can only unmod themselves.",
			Function: func(cl *Client, args []string) string {
				if len(args) > 0 && !cl.IsAdmin {
					return "You can only unmod yourself, not others."
				}

				if len(args) == 0 {
					cl.Unmod()
					return "You have unmodded yourself."
				}

				if err := cl.belongsTo.Unmod(args[0]); err != nil {
					return err.Error()
				}

				return fmt.Sprintf(`%s has been unmodded.`, args[0])
			},
		},

		common.CNKick.String(): Command{
			HelpText: "Kick a user from chat.",
			Function: func(cl *Client, args []string) string {
				if len(args) == 0 {
					return "Missing name to kick."
				}
				return cl.belongsTo.Kick(args[0])
			},
		},

		common.CNBan.String(): Command{
			HelpText: "Ban a user from chat.  They will not be able to re-join chat, but will still be able to view the stream.",
			Function: func(cl *Client, args []string) string {
				if len(args) == 0 {
					return "missing name to ban."
				}
				fmt.Printf("[ban] Attempting to ban %s\n", strings.Join(args, ""))
				return cl.belongsTo.Ban(args[0])
			},
		},

		common.CNUnban.String(): Command{
			HelpText: "Remove a ban on a user.",
			Function: func(cl *Client, args []string) string {
				if len(args) == 0 {
					return "missing name to unban."
				}
				fmt.Printf("[ban] Attempting to unban %s\n", strings.Join(args, ""))

				err := settings.RemoveBan(args[0])
				if err != nil {
					return err.Error()
				}
				return ""
			},
		},
	},

	admin: map[string]Command{
		common.CNMod.String(): Command{
			HelpText: "Grant moderator privilages to a user.",
			Function: func(cl *Client, args []string) string {
				if len(args) == 0 {
					return "Missing user to mod."
				}
				if err := cl.belongsTo.Mod(args[0]); err != nil {
					return err.Error()
				}
				return fmt.Sprintf(`%s has been modded.`, args[0])
			},
		},

		common.CNReloadPlayer.String(): Command{
			HelpText: "Reload the stream player for everybody in chat.",
			Function: func(cl *Client, args []string) string {
				cl.belongsTo.AddCmdMsg(common.CmdRefreshPlayer, nil)
				return "Reloading player for all chatters."
			},
		},

		common.CNReloadEmotes.String(): Command{
			HelpText: "Reload the emotes on the server.",
			Function: func(cl *Client, args []string) string {
				cl.ServerMessage("Reloading emotes")
				num, err := LoadEmotes()
				if err != nil {
					fmt.Printf("Unbale to reload emotes: %s\n", err)
					return fmt.Sprintf("ERROR: %s", err)
				}

				fmt.Printf("Loaded %d emotes\n", num)
				return fmt.Sprintf("Emotes loaded: %d", num)
			},
		},

		common.CNModpass.String(): Command{
			HelpText: "Generate a single-use mod password.",
			Function: func(cl *Client, args []string) string {
				password := cl.belongsTo.generateModPass()
				return "Single use password: " + password
			},
		},
	},
}

func (cc *CommandControl) RunCommand(command string, args []string, sender *Client) string {
	// get correct command from combined commands
	cmd := common.GetFullChatCommand(command)

	// Look for user command
	if userCmd, ok := cc.user[cmd]; ok {
		fmt.Printf("[user] %s /%s %s\n", sender.name, command, strings.Join(args, " "))
		return userCmd.Function(sender, args)
	}

	// Look for mod command
	if modCmd, ok := cc.mod[cmd]; ok {
		if sender.IsMod || sender.IsAdmin {
			fmt.Printf("[mod] %s /%s %s\n", sender.name, command, strings.Join(args, " "))
			return modCmd.Function(sender, args)
		}

		fmt.Printf("[mod REJECTED] %s /%s %s\n", sender.name, command, strings.Join(args, " "))
		return "You are not a mod Jebaited"
	}

	// Look for admin command
	if adminCmd, ok := cc.admin[cmd]; ok {
		if sender.IsAdmin {
			fmt.Printf("[admin] %s /%s %s\n", sender.name, command, strings.Join(args, " "))
			return adminCmd.Function(sender, args)
		}
		fmt.Printf("[admin REJECTED] %s /%s %s\n", sender.name, command, strings.Join(args, " "))
		return "You are not the admin Jebaited"
	}

	// Command not found
	fmt.Printf("[cmd] %s /%s %s\n", sender.name, command, strings.Join(args, " "))
	return "Invalid command."
}

func cmdHelp(cl *Client, args []string) string {
	url := "/help"
	if cl.IsMod {
		url = "/modhelp"
	}

	if cl.IsAdmin {
		url = "/adminhelp"
	}

	return `Opening help in new window.<script>window.open("` + url + `", "_blank", "menubar=0,status=0,toolbar=0,width=300,height=600")</script>`
}

// Return a full HTML page for the help text.  This should probably be rewritten with templates.
func helpPage(ismod, isadmin bool) string {
	if commands == nil {
		return "No commands loaded Jebaited"
	}

	text := []string{}
	appendText := func(group map[string]Command) {
		for key, cmd := range group {
			for _, k := range strings.Split(key, common.CommandNameSeparator) {
				text = append(text, fmt.Sprintf(`<dl class="helptext"><dt>%s</dt><dd>%s</dd></dl>`, k, cmd.HelpText))
			}
		}
	}

	appendText(commands.user)

	if ismod {
		appendText(commands.mod)
	}

	if isadmin {
		appendText(commands.admin)
	}

	// This is ugly
	return `<html><head><title>Help</title><link rel="stylesheet" type="text/css" href="/static/site.css"></head><body>` + strings.Join(text, "") + `</body></html>`
}

// Commands below have more than one invoking command (aliases).

var cmdColor = Command{
	HelpText: "Change user color.",
	Function: func(cl *Client, args []string) string {
		// If the caller is priviledged enough, they can change the color of another user
		if len(args) == 2 && (cl.IsMod || cl.IsAdmin) {
			color := ""
			name := ""
			for _, s := range args {
				if strings.HasPrefix(s, "#") {
					color = s
				} else {
					name = s
				}
			}
			if color == "" {

				fmt.Printf("[color:mod] %s missing color\n", cl.name)
				return "Missing color"
			}

			if err := cl.belongsTo.ForceColorChange(name, color); err != nil {
				return err.Error()
			}
			return fmt.Sprintf("Color changed for user %s to %s\n", name, color)
		}

		// Don't allow an unprivilaged user to change their color if
		// it was changed by a mod
		if cl.IsColorForced {
			fmt.Printf("[color] %s tried to change a forced color\n", cl.name)
			return "You are not allowed to change your color."
		}

		if len(args) == 0 {
			cl.color = randomColor()
			return "Random color chosen: " + cl.color
		}

		// Change the color of the user
		if !colorRegex.MatchString(args[0]) {
			return "To choose a specific color use the format <i>/color #c029ce</i>.  Hex values expected."
		}

		cl.color = args[0]
		fmt.Printf("[color] %s new color: %s\n", cl.name, cl.color)
		return "Color changed successfully."
	},
}

var cmdWhoAmI = Command{
	HelpText: "Shows debug user info",
	Function: func(cl *Client, args []string) string {
		return fmt.Sprintf("Name: %s IsMod: %t IsAdmin: %t", cl.name, cl.IsMod, cl.IsAdmin)
	},
}
