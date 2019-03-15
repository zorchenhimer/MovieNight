package common

type ClientDataType int

const (
	CdMessage ClientDataType = iota // a normal message from the client meant to be broadcast
	CdUsers                         // get a list of users
)

type DataType int

// Data Types
const (
	DTInvalid DataType = iota
	DTChat             // chat message
	DTError            // something went wrong with the previous request
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

type EventType int

// Event Types
const (
	EvJoin EventType = iota
	EvLeave
	EvKick
	EvBan
	EvServerMessage
)

type MessageType int

// Message Types
const (
	MsgChat   MessageType = iota // standard chat
	MsgAction                    // /me command
	MsgServer                    // server message
	MsgError
	MsgNotice // Like MsgServer, but for mods and admins only.
)
