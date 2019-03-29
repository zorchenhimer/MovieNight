package main

import (
	"fmt"
	"html"
	"strings"

	"github.com/zorchenhimer/MovieNight/common"
)

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

var commands = &CommandControl{
	user: map[string]Command{
		common.CNMe.String(): Command{
			HelpText: "Display an action message.",
			Function: func(client *Client, args []string) string {
				if len(args) != 0 {
					client.Me(strings.Join(args, " "))
				}
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

		common.CNColor.String(): Command{
			HelpText: "Change user color.",
			Function: func(cl *Client, args []string) string {
				if len(args) > 2 {
					return "Too many arguments!"
				}

				// If the caller is priviledged enough, they can change the color of another user
				if len(args) == 2 {
					if cl.CmdLevel == common.CmdlUser {
						return "You cannot change someone else's color. PeepoSus"
					}

					name, color := "", ""

					if strings.ToLower(args[0]) == strings.ToLower(args[1]) ||
						(common.IsValidColor(args[0]) && common.IsValidColor(args[1])) {
						return "Name and color are ambiguous. Prefix the name with '@' or color with '#'"
					}

					// Check for explicit name
					if strings.HasPrefix(args[0], "@") {
						name = strings.TrimLeft(args[0], "@")
						color = args[1]
						common.LogDebugln("[color:mod] Found explicit name: ", name)
					} else if strings.HasPrefix(args[1], "@") {
						name = strings.TrimLeft(args[1], "@")
						color = args[0]
						common.LogDebugln("[color:mod] Found explicit name: ", name)

						// Check for explicit color
					} else if strings.HasPrefix(args[0], "#") {
						name = strings.TrimPrefix(args[1], "@") // this shouldn't be needed, but just in case.
						color = args[0]
						common.LogDebugln("[color:mod] Found explicit color: ", color)
					} else if strings.HasPrefix(args[1], "#") {
						name = strings.TrimPrefix(args[0], "@") // this shouldn't be needed, but just in case.
						color = args[1]
						common.LogDebugln("[color:mod] Found explicit color: ", color)

						// Guess
					} else if common.IsValidColor(args[0]) {
						name = strings.TrimPrefix(args[1], "@")
						color = args[0]
						common.LogDebugln("[color:mod] Guessed name: ", name, " and color: ", color)
					} else if common.IsValidColor(args[1]) {
						name = strings.TrimPrefix(args[0], "@")
						color = args[1]
						common.LogDebugln("[color:mod] Guessed name: ", name, " and color: ", color)
					}

					if name == "" {
						return "Cannot determine name.  Prefix name with @."
					}
					if color == "" {
						return "Cannot determine color.  Prefix name with @."
					}

					if color == "" {
						common.LogInfof("[color:mod] %s missing color\n", cl.name)
						return "Missing color"
					}

					if name == "" {
						common.LogInfof("[color:mod] %s missing name\n", cl.name)
						return "Missing name"
					}

					if err := cl.belongsTo.ForceColorChange(name, color); err != nil {
						return err.Error()
					}
					return fmt.Sprintf("Color changed for user %s to %s\n", name, color)
				}

				// Don't allow an unprivilaged user to change their color if
				// it was changed by a mod
				if cl.IsColorForced {
					common.LogInfof("[color] %s tried to change a forced color\n", cl.name)
					return "You are not allowed to change your color."
				}

				if len(args) == 0 {
					cl.setColor(common.RandomColor())
					return "Random color chosen: " + cl.color
				}

				// Change the color of the user
				if !common.IsValidColor(args[0]) {
					return "To choose a specific color use the format <i>/color #c029ce</i>.  Hex values expected."
				}

				err := cl.setColor(args[0])
				if err != nil {
					common.LogErrorf("[color] could not send color update to client: %v\n", err)
				}

				common.LogInfof("[color] %s new color: %s\n", cl.name, cl.color)
				return "Color changed successfully."
			},
		},

		common.CNWhoAmI.String(): Command{
			HelpText: "Shows debug user info",
			Function: func(cl *Client, args []string) string {
				return fmt.Sprintf("Name: %s IsMod: %t IsAdmin: %t",
					cl.name,
					cl.CmdLevel >= common.CmdlMod,
					cl.CmdLevel == common.CmdlAdmin)
			},
		},

		common.CNAuth.String(): Command{
			HelpText: "Authenticate to admin",
			Function: func(cl *Client, args []string) string {
				if cl.CmdLevel == common.CmdlAdmin {
					return "You are already authenticated."
				}

				pw := html.UnescapeString(strings.Join(args, " "))

				if settings.AdminPassword == pw {
					cl.CmdLevel = common.CmdlAdmin
					cl.belongsTo.AddModNotice(cl.name + " used the admin password")
					common.LogInfof("[auth] %s used the admin password\n", cl.name)
					return "Admin rights granted."
				}

				if cl.belongsTo.redeemModPass(pw) {
					cl.CmdLevel = common.CmdlMod
					cl.belongsTo.AddModNotice(cl.name + " used a mod password")
					common.LogInfof("[auth] %s used a mod password\n", cl.name)
					return "Moderator privileges granted."
				}

				cl.belongsTo.AddModNotice(cl.name + " attempted to auth without success")
				common.LogInfof("[auth] %s gave an invalid password\n", cl.name)
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

		common.CNNick.String(): Command{
			HelpText: "Change display name",
			Function: func(cl *Client, args []string) string {
				if len(args) == 0 {
					return "Missing name to change to."
				}

				newName := strings.TrimLeft(args[0], "@")
				oldName := cl.name
				forced := false

				// Two arguments to force a name change on another user: `/nick OldName NewName`
				if len(args) == 2 {
					if cl.CmdLevel != common.CmdlAdmin {
						return "Only admins can do that PeepoSus"
					}

					oldName = strings.TrimLeft(args[0], "@")
					newName = strings.TrimLeft(args[1], "@")
					forced = true
				}

				if len(args) == 1 && cl.IsNameForced && cl.CmdLevel != common.CmdlAdmin {
					return "You cannot change your name once it has been changed by an admin."
				}

				err := cl.belongsTo.changeName(oldName, newName, forced)
				if err != nil {
					return "Unable to change name: " + err.Error()
				}

				return ""
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
				svmsg := formatLinks(strings.Join(common.ParseEmotesArray(args), " "))
				cl.belongsTo.AddModNotice("Server message from " + cl.name)
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
					return "Title cleared"
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

				title = strings.TrimSpace(title)
				link = strings.TrimSpace(link)

				if len(title) > settings.TitleLength {
					return fmt.Sprintf("Title too long (%d/%d)", len(title), settings.TitleLength)
				}

				// Send a notice to the mods and admins
				if len(link) == 0 {
					cl.belongsTo.AddModNotice(cl.name + " set the playing title to '" + title + "' with no link")
				} else {
					cl.belongsTo.AddModNotice(cl.name + " set the playing title to '" + title + "' with link '" + link + "'")
				}

				cl.belongsTo.SetPlaying(title, link)
				return ""
			},
		},

		common.CNUnmod.String(): Command{
			HelpText: "Revoke a user's moderator privilages.  Moderators can only unmod themselves.",
			Function: func(cl *Client, args []string) string {
				if len(args) > 0 && cl.CmdLevel != common.CmdlAdmin && cl.name != args[0] {
					return "You can only unmod yourself, not others."
				}

				if len(args) == 0 || (len(args) == 1 && strings.TrimLeft(args[0], "@") == cl.name) {
					cl.Unmod()
					cl.belongsTo.AddModNotice(cl.name + " has unmodded themselves")
					return "You have unmodded yourself."
				}
				name := strings.TrimLeft(args[0], "@")

				if err := cl.belongsTo.Unmod(name); err != nil {
					return err.Error()
				}

				cl.belongsTo.AddModNotice(cl.name + " has unmodded " + name)
				return ""
			},
		},

		common.CNKick.String(): Command{
			HelpText: "Kick a user from chat.",
			Function: func(cl *Client, args []string) string {
				if len(args) == 0 {
					return "Missing name to kick."
				}
				return cl.belongsTo.Kick(strings.TrimLeft(args[0], "@"))
			},
		},

		common.CNBan.String(): Command{
			HelpText: "Ban a user from chat.  They will not be able to re-join chat, but will still be able to view the stream.",
			Function: func(cl *Client, args []string) string {
				if len(args) == 0 {
					return "missing name to ban."
				}

				name := strings.TrimLeft(args[0], "@")
				common.LogInfof("[ban] Attempting to ban %s\n", name)
				return cl.belongsTo.Ban(name)
			},
		},

		common.CNUnban.String(): Command{
			HelpText: "Remove a ban on a user.",
			Function: func(cl *Client, args []string) string {
				if len(args) == 0 {
					return "missing name to unban."
				}
				name := strings.TrimLeft(args[0], "@")
				common.LogInfof("[ban] Attempting to unban %s\n", name)

				err := settings.RemoveBan(name)
				if err != nil {
					return err.Error()
				}
				cl.belongsTo.AddModNotice(cl.name + " has unbanned " + name)
				return ""
			},
		},

		common.CNPurge.String(): Command{
			HelpText: "Purge the chat.",
			Function: func(cl *Client, args []string) string {
				common.LogInfoln("[purge] clearing chat")
				cl.belongsTo.AddCmdMsg(common.CmdPurgeChat, nil)
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

				name := strings.TrimLeft(args[0], "@")
				if err := cl.belongsTo.Mod(name); err != nil {
					return err.Error()
				}
				cl.belongsTo.AddModNotice(cl.name + " has modded " + name)
				return ""
			},
		},

		common.CNReloadPlayer.String(): Command{
			HelpText: "Reload the stream player for everybody in chat.",
			Function: func(cl *Client, args []string) string {
				cl.belongsTo.AddModNotice(cl.name + " has modded forced a player reload")
				cl.belongsTo.AddCmdMsg(common.CmdRefreshPlayer, nil)
				return "Reloading player for all chatters."
			},
		},

		common.CNReloadEmotes.String(): Command{
			HelpText: "Reload the emotes on the server.",
			Function: func(cl *Client, args []string) string {
				cl.SendServerMessage("Reloading emotes")
				num, err := common.LoadEmotes()
				if err != nil {
					common.LogErrorf("Unbale to reload emotes: %s\n", err)
					return fmt.Sprintf("ERROR: %s", err)
				}

				cl.belongsTo.AddModNotice(cl.name + " has reloaded emotes")
				common.LogInfof("Loaded %d emotes\n", num)
				return fmt.Sprintf("Emotes loaded: %d", num)
			},
		},

		common.CNModpass.String(): Command{
			HelpText: "Generate a single-use mod password.",
			Function: func(cl *Client, args []string) string {
				cl.belongsTo.AddModNotice(cl.name + " generated a mod password")
				password := cl.belongsTo.generateModPass()
				return "Single use password: " + password
			},
		},

		common.CNIP.String(): Command{
			HelpText: "List users and IP in the server console.  Requires logging level to be set to info or above.",
			Function: func(cl *Client, args []string) string {
				cl.belongsTo.clientsMtx.Lock()
				common.LogInfoln("Clients:")
				for uuid, client := range cl.belongsTo.clients {
					common.LogInfof("  [%s] %s %s\n", uuid, client.name, client.conn.Host())
				}

				common.LogInfoln("TmpConn:")
				for uuid, conn := range cl.belongsTo.tempConn {
					common.LogInfof("  [%s] %s\n", uuid, conn.Host())
				}
				cl.belongsTo.clientsMtx.Unlock()
				return "see console for output"
			},
		},

		common.CNAddEmotes.String(): Command{
			HelpText: "Add emotes from a given twitch channel.",
			Function: func(cl *Client, args []string) string {
				// Fire this off in it's own goroutine so the client doesn't
				// block waiting for the emote download to finish.
				go func() {

					// Pretty sure this breaks on partial downloads (eg, one good channel and one non-existant)
					_, err := GetEmotes(args)
					if err != nil {
						cl.SendChatData(common.NewChatMessage("", "",
							err.Error(),
							common.CmdlUser, common.MsgCommandResponse))
						return
					}

					// reload emotes now that new ones were added
					_, err = common.LoadEmotes()
					if err != nil {
						cl.SendChatData(common.NewChatMessage("", "",
							err.Error(),
							common.CmdlUser, common.MsgCommandResponse))
						return
					}

					cl.belongsTo.AddModNotice(cl.name + " has added emotes from the following channels: " + strings.Join(args, ", "))
				}()
				return "Emote download initiated for the following channels: " + strings.Join(args, ", ")
			},
		},
	},
}

func (cc *CommandControl) RunCommand(command string, args []string, sender *Client) string {
	// get correct command from combined commands
	cmd := common.GetFullChatCommand(command)

	// Look for user command
	if userCmd, ok := cc.user[cmd]; ok {
		common.LogInfof("[user] %s /%s %s\n", sender.name, command, strings.Join(args, " "))
		return userCmd.Function(sender, args)
	}

	// Look for mod command
	if modCmd, ok := cc.mod[cmd]; ok {
		if sender.CmdLevel >= common.CmdlMod {
			common.LogInfof("[mod] %s /%s %s\n", sender.name, command, strings.Join(args, " "))
			return modCmd.Function(sender, args)
		}

		common.LogInfof("[mod REJECTED] %s /%s %s\n", sender.name, command, strings.Join(args, " "))
		return "You are not a mod Jebaited"
	}

	// Look for admin command
	if adminCmd, ok := cc.admin[cmd]; ok {
		if sender.CmdLevel == common.CmdlAdmin {
			common.LogInfof("[admin] %s /%s %s\n", sender.name, command, strings.Join(args, " "))
			return adminCmd.Function(sender, args)
		}
		common.LogInfof("[admin REJECTED] %s /%s %s\n", sender.name, command, strings.Join(args, " "))
		return "You are not the admin Jebaited"
	}

	// Command not found
	common.LogInfof("[cmd] %s /%s %s\n", sender.name, command, strings.Join(args, " "))
	return "Invalid command."
}

func cmdHelp(cl *Client, args []string) string {
	url := "/help"

	if cl.CmdLevel >= common.CmdlMod {
		url += "?mod=1"
	}

	if cl.CmdLevel == common.CmdlAdmin {
		url += "&admin=1"
	}

	cl.SendChatData(common.NewChatCommand(common.CmdHelp, []string{url}))
	return `Opening help in new window.`
}

func getHelp(lvl common.CommandLevel) map[string]string {
	var cmdList map[string]Command
	switch lvl {
	case common.CmdlUser:
		cmdList = commands.user
	case common.CmdlMod:
		cmdList = commands.mod
	case common.CmdlAdmin:
		cmdList = commands.admin
	}

	helptext := map[string]string{}
	for name, cmd := range cmdList {
		helptext[name] = cmd.HelpText
	}
	return helptext
}
