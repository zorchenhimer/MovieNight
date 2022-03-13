package main

import (
	"fmt"
	"html"
	"strings"
	"time"

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

type CommandFunction func(client *Client, args []string) (string, error)

var commands = &CommandControl{
	user: map[string]Command{
		common.CNMe.String(): {
			HelpText: "Display an action message.",
			Function: func(client *Client, args []string) (string, error) {
				if len(args) != 0 {
					client.Me(strings.Join(args, " "))
					return "", nil
				}
				return "", newChatError("Missing a message")
			},
		},

		common.CNHelp.String(): {
			HelpText: "This help text.",
			Function: cmdHelp,
		},

		common.CNEmotes.String(): {
			HelpText: "Display a list of available emotes.",
			Function: func(client *Client, args []string) (string, error) {
				err := client.SendChatData(common.NewChatCommand(common.CmdEmotes, []string{"/emotes"}))
				if err != nil {
					return "", fmt.Errorf("could not send /emotes command: %w", err)
				}
				return "Opening emote list in new window.", nil
			},
		},

		common.CNCount.String(): {
			HelpText: "Display number of users in chat.",
			Function: func(client *Client, args []string) (string, error) {
				return fmt.Sprintf("Users in chat: %d", client.belongsTo.UserCount()), nil
			},
		},

		common.CNColor.String(): {
			HelpText: "Change user color.",
			Function: func(cl *Client, args []string) (string, error) {
				if len(args) > 2 {
					return "", newChatError("Too many arguments!")
				}

				// If the caller is privileged enough, they can change the color of another user
				if len(args) == 2 {
					if cl.CmdLevel == common.CmdlUser {
						return "", newChatError("You cannot change someone else's color. PeepoSus")
					}

					name, color := "", ""

					if strings.EqualFold(args[0], args[1]) ||
						(common.IsValidColor(args[0]) && common.IsValidColor(args[1])) {
						return "", newChatError("Name and color are ambiguous. Prefix the name with '@' or color with '#'")
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
						return "", newChatError("Cannot determine name.  Prefix name with @.")
					}
					if color == "" {
						return "", newChatError("Cannot determine color.  Prefix name with @.")
					}

					if color == "" {
						common.LogInfof("[color:mod] %s missing color\n", cl.name)
						return "", newChatError("Missing color")
					}

					if name == "" {
						common.LogInfof("[color:mod] %s missing name\n", cl.name)
						return "", newChatError("Missing name")
					}

					if err := cl.belongsTo.ForceColorChange(name, color); err != nil {
						return "", err
					}
					return fmt.Sprintf("Color changed for user %s to %s\n", name, color), nil
				}

				// Don't allow an unprivileged user to change their color if
				// it was changed by a mod
				if cl.IsColorForced {
					common.LogInfof("[color] %s tried to change a forced color\n", cl.name)
					return "", newChatError("You are not allowed to change your color.")
				}

				if time.Now().Before(cl.nextColor) && cl.CmdLevel == common.CmdlUser {
					return "", newChatError("Slow down. You can change your color in %0.0f seconds.", time.Until(cl.nextColor).Seconds())
				}

				if len(args) == 0 {
					err := cl.setColor(common.RandomColor())
					if err != nil {
						common.LogInfof("[color] failed to set random color for %s: %v\n", cl.name, err)
						return "", newChatError("Could not set random color")
					}
					return "Random color chosen: " + cl.color, nil
				}

				// Change the color of the user
				if !common.IsValidColor(args[0]) {
					return "", newChatError("To choose a specific color use the format <i>/color #c029ce</i>.  Hex values expected.")
				}

				cl.nextColor = time.Now().Add(time.Second * settings.RateLimitColor)

				err := cl.setColor(args[0])
				if err != nil {
					common.LogErrorf("[color] could not send color update to client: %v\n", err)
				}

				common.LogInfof("[color] %s new color: %s\n", cl.name, cl.color)
				return "Color changed successfully.", nil
			},
		},

		common.CNWhoAmI.String(): {
			HelpText: "Shows debug user info",
			Function: func(cl *Client, args []string) (string, error) {
				return fmt.Sprintf("Name: %s IsMod: %t IsAdmin: %t",
					cl.name,
					cl.CmdLevel >= common.CmdlMod,
					cl.CmdLevel == common.CmdlAdmin), nil
			},
		},

		common.CNAuth.String(): {
			HelpText: "Authenticate to admin",
			Function: func(cl *Client, args []string) (string, error) {
				if cl.CmdLevel == common.CmdlAdmin {
					return "", newChatError("You are already authenticated.")
				}

				// TODO: handle back off policy
				if time.Now().Before(cl.nextAuth) {
					cl.nextAuth = time.Now().Add(time.Second * settings.RateLimitAuth)
					return "", newChatError("Slow down.")
				}
				cl.authTries++ // this isn't used yet
				cl.nextAuth = time.Now().Add(time.Second * settings.RateLimitAuth)

				pw := html.UnescapeString(strings.Join(args, " "))

				if settings.AdminPassword == pw {
					cl.CmdLevel = common.CmdlAdmin
					cl.belongsTo.AddModNotice(cl.name + " used the admin password")
					common.LogInfof("[auth] %s used the admin password\n", cl.name)
					return "Admin rights granted.", nil
				}

				if cl.belongsTo.redeemModPass(pw) {
					cl.CmdLevel = common.CmdlMod
					cl.belongsTo.AddModNotice(cl.name + " used a mod password")
					common.LogInfof("[auth] %s used a mod password\n", cl.name)
					return "Moderator privileges granted.", nil
				}

				cl.belongsTo.AddModNotice(cl.name + " attempted to auth without success")
				common.LogInfof("[auth] %s gave an invalid password\n", cl.name)
				return "", newChatError("Invalid password.")
			},
		},

		common.CNUsers.String(): {
			HelpText: "Show a list of users in chat",
			Function: func(cl *Client, args []string) (string, error) {
				names := cl.belongsTo.GetNames()
				formatNames := func(names []string) []string {
					result := make([]string, len(names))
					for _, name := range names {
						if strings.HasPrefix(name, "@") {
							result = append(result, name)
						} else {
							result = append(result, "@"+name)
						}
					}
					return result
				}
				return strings.Join(formatNames(names), " "), nil
			},
		},

		common.CNNick.String(): {
			HelpText: "Change display name",
			Function: func(cl *Client, args []string) (string, error) {
				if time.Now().Before(cl.nextNick) && cl.CmdLevel == common.CmdlUser {
					//cl.nextNick = time.Now().Add(time.Second * settings.RateLimitNick)
					return "", newChatError("Slow down. You can change your nick in %0.0f seconds.", time.Until(cl.nextNick).Seconds())
				}
				cl.nextNick = time.Now().Add(time.Second * settings.RateLimitNick)

				if len(args) == 0 {
					return "", newChatError("Missing name to change to.")
				}

				newName := strings.TrimLeft(args[0], "@")
				oldName := cl.name
				forced := false

				// Two arguments to force a name change on another user: `/nick OldName NewName`
				if len(args) == 2 {
					if cl.CmdLevel == common.CmdlUser {
						return "", newChatError("Only admins and mods can do that PeepoSus")
					}

					oldName = strings.TrimLeft(args[0], "@")
					newName = strings.TrimLeft(args[1], "@")
					forced = true
				}

				if len(args) == 1 && cl.IsNameForced && cl.CmdLevel != common.CmdlAdmin {
					return "", newChatError("You cannot change your name once it has been changed by an admin.")
				}

				err := cl.belongsTo.changeName(oldName, newName, forced)
				if err != nil {
					return "", newChatError("Unable to change name: " + err.Error())
				}

				return "", nil
			},
		},

		common.CNStats.String(): {
			HelpText: "Show some stats for stream.",
			Function: func(cl *Client, args []string) (string, error) {
				cl.belongsTo.clientsMtx.Lock()
				users := len(cl.belongsTo.clients)
				cl.belongsTo.clientsMtx.Unlock()

				// Just print max users and time alive here
				return fmt.Sprintf("Current users in chat: <b>%d</b><br />Max users in chat: <b>%d</b><br />Server uptime: <b>%s</b><br />Stream uptime: <b>%s</b><br />Viewers: <b>%d</b><br />Max Viewers: <b>%d</b>",
					users,
					stats.getMaxUsers(),
					time.Since(stats.start),
					stats.getStreamLength(),
					stats.getViewerCount(),
					stats.getMaxViewerCount(),
				), nil
			},
		},

		common.CNPin.String(): {
			HelpText: "Display the current room access type and pin/password (if applicable).",
			Function: func(cl *Client, args []string) (string, error) {
				switch settings.RoomAccess {
				case AccessPin:
					return "Room is secured via PIN.  Current PIN: " + settings.RoomAccessPin, nil
				case AccessRequest:
					return "Room is secured via access requests.  Users must request to be granted access.", nil
				}
				return "Room is open access.  Anybody can join.", nil
			},
		},
	},

	mod: map[string]Command{
		common.CNSv.String(): {
			HelpText: "Send a server announcement message.  It will show up red with a border in chat.",
			Function: func(cl *Client, args []string) (string, error) {
				if len(args) == 0 {
					return "", newChatError("Missing message")
				}
				svmsg := formatLinks(strings.Join(common.ParseEmotesArray(args), " "))
				cl.belongsTo.AddModNotice("Server message from " + cl.name)
				cl.belongsTo.AddMsg(cl, false, true, svmsg)
				return "", nil
			},
		},

		common.CNPlaying.String(): {
			HelpText: "Set the title text and info link.",
			Function: func(cl *Client, args []string) (string, error) {
				// Clear/hide title if sent with no arguments.
				if len(args) == 0 {
					cl.belongsTo.ClearPlaying()
					return "Title cleared", nil
				}
				link := ""
				title := ""

				// pick out the link (can be anywhere, as long as there are no spaces).
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
					return "", newChatError("Title too long (%d/%d)", len(title), settings.TitleLength)
				}

				// Send a notice to the mods and admins
				if len(link) == 0 {
					cl.belongsTo.AddModNotice(cl.name + " set the playing title to '" + title + "' with no link")
				} else {
					cl.belongsTo.AddModNotice(cl.name + " set the playing title to '" + title + "' with link '" + link + "'")
				}

				cl.belongsTo.SetPlaying(title, link)
				return "", nil
			},
		},

		common.CNUnmod.String(): {
			HelpText: "Revoke a user's moderator privilages.  Moderators can only unmod themselves.",
			Function: func(cl *Client, args []string) (string, error) {
				if len(args) > 0 && cl.CmdLevel != common.CmdlAdmin && cl.name != args[0] {
					return "", newChatError("You can only unmod yourself, not others.")
				}

				if len(args) == 0 || (len(args) == 1 && strings.TrimLeft(args[0], "@") == cl.name) {
					cl.Unmod()
					cl.belongsTo.AddModNotice(cl.name + " has unmodded themselves")
					return "You have unmodded yourself.", nil
				}
				name := strings.TrimLeft(args[0], "@")

				if err := cl.belongsTo.Unmod(name); err != nil {
					return "", err
				}

				cl.belongsTo.AddModNotice(cl.name + " has unmodded " + name)
				return "", nil
			},
		},

		common.CNKick.String(): {
			HelpText: "Kick a user from chat.",
			Function: func(cl *Client, args []string) (string, error) {
				if len(args) == 0 {
					return "", newChatError("Missing name to kick.")
				}
				return "", cl.belongsTo.Kick(strings.TrimLeft(args[0], "@"))
			},
		},

		common.CNBan.String(): {
			HelpText: "Ban a user from chat.  They will not be able to re-join chat, but will still be able to view the stream.",
			Function: func(cl *Client, args []string) (string, error) {
				if len(args) == 0 {
					return "", newChatError("missing name to ban.")
				}

				name := strings.TrimLeft(args[0], "@")
				common.LogInfof("[ban] Attempting to ban %s\n", name)
				return "", cl.belongsTo.Ban(name)
			},
		},

		common.CNUnban.String(): {
			HelpText: "Remove a ban on a user.",
			Function: func(cl *Client, args []string) (string, error) {
				if len(args) == 0 {
					return "", newChatError("missing name to unban.")
				}
				name := strings.TrimLeft(args[0], "@")
				common.LogInfof("[ban] Attempting to unban %s\n", name)

				err := settings.RemoveBan(name)
				if err != nil {
					return "", err
				}
				cl.belongsTo.AddModNotice(cl.name + " has unbanned " + name)
				return "", nil
			},
		},

		common.CNPurge.String(): {
			HelpText: "Purge the chat.",
			Function: func(cl *Client, args []string) (string, error) {
				common.LogInfoln("[purge] clearing chat")
				cl.belongsTo.AddCmdMsg(common.CmdPurgeChat, nil)
				return "", nil
			},
		},
	},

	admin: map[string]Command{
		common.CNMod.String(): {
			HelpText: "Grant moderator privilages to a user.",
			Function: func(cl *Client, args []string) (string, error) {
				if len(args) == 0 {
					return "", newChatError("Missing user to mod.")
				}

				name := strings.TrimLeft(args[0], "@")
				if err := cl.belongsTo.Mod(name); err != nil {
					return "", err
				}
				cl.belongsTo.AddModNotice(cl.name + " has modded " + name)
				return "", nil
			},
		},

		common.CNReloadPlayer.String(): {
			HelpText: "Reload the stream player for everybody in chat.",
			Function: func(cl *Client, args []string) (string, error) {
				cl.belongsTo.AddModNotice(cl.name + " has modded forced a player reload")
				cl.belongsTo.AddCmdMsg(common.CmdRefreshPlayer, nil)
				return "Reloading player for all chatters.", nil
			},
		},

		common.CNReloadEmotes.String(): {
			HelpText: "Reload the emotes on the server.",
			Function: func(cl *Client, args []string) (string, error) {
				go commandReloadEmotes(cl)
				return "Reloading emotes...", nil
			},
		},

		common.CNModpass.String(): {
			HelpText: "Generate a single-use mod password.",
			Function: func(cl *Client, args []string) (string, error) {
				cl.belongsTo.AddModNotice(cl.name + " generated a mod password")
				password := cl.belongsTo.generateModPass()
				return "Single use password: " + password, nil
			},
		},

		common.CNRoomAccess.String(): {
			HelpText: "Change the room access type.",
			Function: func(cl *Client, args []string) (string, error) {
				// Print current access type if no arguments given
				if len(args) == 0 {
					return "Current room access type: " + string(settings.RoomAccess), nil
				}

				switch AccessMode(strings.ToLower(args[0])) {
				case AccessOpen:
					settings.RoomAccess = AccessOpen
					common.LogInfoln("[access] Room set to open")
					return "Room access set to open", nil

				case AccessPin:
					// A pin/password was provided, use it.
					if len(args) == 2 {
						// TODO: make this a bit more robust.  Currently, only accepts a single word as a pin/password
						settings.RoomAccessPin = args[1]

						// A pin/password was not provided, generate a new one.
					} else {
						_, err := settings.generateNewPin()
						if err != nil {
							common.LogErrorln("Error generating new access pin: ", err.Error())
							return "", newChatError("Unable to generate a new pin, access unchanged: " + err.Error())
						}
					}
					settings.RoomAccess = AccessPin
					common.LogInfoln("[access] Room set to pin: " + settings.RoomAccessPin)
					return "Room access set to Pin: " + settings.RoomAccessPin, nil

				case AccessRequest:
					settings.RoomAccess = AccessRequest
					common.LogInfoln("[access] Room set to request")
					return "Room access set to request. WARNING: this isn't implemented yet.", nil

				default:
					return "", newChatError("Invalid access mode")
				}
			},
		},

		common.CNIP.String(): {
			HelpText: "List users and IP in the server console.  Requires logging level to be set to info or above.",
			Function: func(cl *Client, args []string) (string, error) {
				cl.belongsTo.clientsMtx.Lock()
				common.LogInfoln("Clients:")
				for id, client := range cl.belongsTo.clients {
					common.LogInfof("  [%d] %s %s\n", id, client.name, client.conn.Host())
				}
				cl.belongsTo.clientsMtx.Unlock()
				return "see console for output", nil
			},
		},

		common.CNAddEmotes.String(): {
			HelpText: "Add emotes from a given twitch channel.",
			Function: func(cl *Client, args []string) (string, error) {
				// Fire this off in it's own goroutine so the client doesn't
				// block waiting for the emote download to finish.
				go func() {

					// Pretty sure this breaks on partial downloads (eg, one good channel and one non-existent)
					err := getEmotes(args)
					if err != nil {
						err = cl.SendChatData(common.NewChatMessage("", "", err.Error(), common.CmdlUser, common.MsgCommandResponse))
						if err != nil {
							common.LogErrorln(err)
						}
						return
					}

					// If the emotes were able to be downloaded, add the channels to settings
					err = settings.AddApprovedEmotes(args)
					if err != nil {
						common.LogErrorf("could not add approved emotes: %v\n", err)
					}

					// reload emotes now that new ones were added
					err = loadEmotes()
					if err != nil {
						err = cl.SendChatData(common.NewChatMessage("", "", err.Error(), common.CmdlUser, common.MsgCommandResponse))
						if err != nil {
							common.LogErrorln(err)
						}
						return
					}

					cl.belongsTo.AddModNotice(cl.name + " has added emotes from the following channels: " + strings.Join(args, ", "))

					commandReloadEmotes(cl)
				}()
				return "Emote download initiated for the following channels: " + strings.Join(args, ", "), nil
			},
		},
	},
}

func (cc *CommandControl) RunCommand(command string, args []string, sender *Client) (string, error) {
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
		return "", newChatError("You are not a mod Jebaited")
	}

	// Look for admin command
	if adminCmd, ok := cc.admin[cmd]; ok {
		if sender.CmdLevel == common.CmdlAdmin {
			common.LogInfof("[admin] %s /%s %s\n", sender.name, command, strings.Join(args, " "))
			return adminCmd.Function(sender, args)
		}
		common.LogInfof("[admin REJECTED] %s /%s %s\n", sender.name, command, strings.Join(args, " "))
		return "", newChatError("You are not the admin Jebaited")
	}

	// Command not found
	common.LogInfof("[cmd|error] %s /%s %s\n", sender.name, command, strings.Join(args, " "))
	return "", newChatError("Invalid command.")
}

func cmdHelp(cl *Client, args []string) (string, error) {
	url := "/help"

	if cl.CmdLevel >= common.CmdlMod {
		url += "?mod=1"
	}

	if cl.CmdLevel == common.CmdlAdmin {
		url += "&admin=1"
	}

	err := cl.SendChatData(common.NewChatCommand(common.CmdHelp, []string{url}))
	if err != nil {
		common.LogErrorf("Could not send open window command: %v\n", err)
		return "", newChatError("Could not send open window command")
	}
	return `Opening help in new window.`, nil
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

func commandReloadEmotes(cl *Client) {
	err := cl.SendServerMessage("Reloading emotes")
	if err != nil {
		common.LogErrorf("Could not send reloading emotes server message: %v\n", err)
		return
	}

	err = loadEmotes()
	if err != nil {
		common.LogErrorf("Unbale to reload emotes: %s\n", err)
		err = cl.SendChatData(common.NewChatMessage("", "", err.Error(), common.CmdlUser, common.MsgCommandResponse))
		if err != nil {
			common.LogErrorln(err)
		}

		return
	}

	cl.belongsTo.AddChatMsg(common.NewChatHiddenMessage(common.CdEmote, common.Emotes))
	cl.belongsTo.AddModNotice(cl.name + " has reloaded emotes")

	num := len(common.Emotes)
	common.LogInfof("Loaded %d emotes\n", num)
	cl.belongsTo.AddModNotice(fmt.Sprintf("%s reloaded %d emotes.", cl.name, num))
}
