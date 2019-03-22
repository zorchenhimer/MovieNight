package common

type ClientDataType int

// Data types for communicating with the client
const (
	CdMessage ClientDataType = iota // a normal message from the client meant to be broadcast
	CdUsers                         // get a list of users
	CdPing                          // ping the server to keep the connection alive
	CdHelp                          // tells server to send help data again for buttons
)

type DataType int

// Data types for command messages
const (
	DTInvalid DataType = iota
	DTChat             // chat message
	DTCommand          // non-chat function
	DTEvent            // join/leave/kick/ban events
	DTClient           // a message coming from the client
	DTHidden           // a message that is purely instruction and data, not shown to user
)

type CommandType int

// Command Types
const (
	CmdPlaying CommandType = iota
	CmdRefreshPlayer
	CmdPurgeChat
	CmdHelp
)

type CommandLevel int

// Command access levels
const (
	CmdUser CommandLevel = iota
	CmdMod
	CmdAdmin
)

type EventType int

// Event Types
const (
	EvJoin EventType = iota
	EvLeave
	EvKick
	EvBan
	EvServerMessage
	EvNameChange
	EvNameChangeForced
)

type MessageType int

// Message Types
const (
	MsgChat            MessageType = iota // standard chat
	MsgAction                             // /me command
	MsgServer                             // server message
	MsgError                              // something went wrong
	MsgNotice                             // Like MsgServer, but for mods and admins only.
	MsgCommandResponse                    // The response to a command
)
