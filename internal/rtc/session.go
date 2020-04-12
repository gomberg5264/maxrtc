package rtc

import (
	"github.com/pion/webrtc/v2"
	"github.com/rriverak/gogo/internal/gst"
)

//Session is a GroupVideo Call
type Session struct {
	ID            string        `json:"ID"`
	API           *webrtc.API   `json:"-"`
	Codec         string        `json:"Codec"`
	VideoPipeline *gst.Pipeline `json:"-"`
	AudioPipeline *gst.Pipeline `json:"-"`
	Users         []User        `json:"Users"`
}

//Start a Session with new Parameters
func (s *Session) Start() {
	// Create Pipeline Channel
	chans := []string{}
	for _, usr := range s.Users {
		chans = append(chans, usr.ID)
	}

	// Create GStreamer Pipeline
	s.VideoPipeline = gst.CreateVideoMixerPipeline(s.Codec, chans)

	// Create GStreamer Pipeline
	s.AudioPipeline = gst.CreateAudioMixerPipeline(webrtc.Opus, chans)

	for _, usr := range s.Users {
		s.VideoPipeline.AddOutputTrack(usr.VideOutput())
		s.AudioPipeline.AddOutputTrack(usr.AudioOutput())
	}

	// Start Pipeline output
	s.VideoPipeline.Start()
	s.AudioPipeline.Start()
}

//Stop a Session
func (s *Session) Stop() {
	// Stop Running Pipeline
	if s.VideoPipeline != nil {
		// Set Locking
		s.VideoPipeline.Stop()
		s.VideoPipeline = nil
	}
	if s.AudioPipeline != nil {
		// Set Locking
		s.AudioPipeline.Stop()
		s.AudioPipeline = nil
	}

}

//Restart a Session with new Parameters
func (s *Session) Restart() {
	s.Stop()
	s.Start()
}

// CreateUser in the Session
func (s *Session) CreateUser(name string, peerConnectionConfig webrtc.Configuration, offer webrtc.SessionDescription) (*User, error) {
	// Create New User with Peer
	newUser, err := NewUser(name, peerConnectionConfig, offer)
	if err != nil {
		return nil, err
	}
	// Register Users RemoteTrack with Session
	newUser.Peer.OnTrack(newUser.RemoteTrackHandler(s))
	// Register Session Auto-Leave on Timeout
	newUser.Peer.OnConnectionStateChange(newUser.OnUserConnectionStateChangedHandler(s))
	s.Codec = newUser.Codec
	// Register DataChannels
	newUser.Peer.OnDataChannel(newUser.OnUserDataChannel(s))
	// Add User to Session
	s.RegisterUserDataChannel(newUser)

	s.AddUser(*newUser)
	return newUser, nil
}

//AddUser to Session and restart Pipeline
func (s *Session) AddUser(newUser User) {
	// Add user to Collection
	if s.Users == nil {
		s.Users = make([]User, 0)
	}
	s.Users = append(s.Users, newUser)

	// Restart Session Pipeline
	s.Restart()
}

// RegisterUserDataChannel sadass
func (s *Session) RegisterUserDataChannel(user *User) error {
	// DataChannel for Session Created
	dc, err := s.createSessionDataChannel(user)
	if err != nil {
		return err
	}
	user.DataChannels["session"] = dc
	return nil
}

func (s *Session) createSessionDataChannel(user *User) (*webrtc.DataChannel, error) {
	var id uint16 = 1
	negotiated := false
	opt := webrtc.DataChannelInit{Negotiated: &negotiated, ID: &id}
	dc, err := user.Peer.CreateDataChannel("session", &opt)
	if err != nil {
		Logger.Error(err)
	}
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		message := string(msg.Data)
		Logger.Infof("User => %s send a Message on Session Channel => '%s'", user.Name, message)
	})
	dc.OnOpen(func() {
		err = dc.SendText("OK!")
		if err != nil {
			Logger.Error(err)
		}
	})
	return dc, nil
}

//RemoveUser from with ID from Session and restart Pipeline
func (s *Session) RemoveUser(usrID string) {
	if s.Users == nil {
		s.Users = make([]User, 0)
	}
	tmpUsers := []User{}
	for _, usr := range s.Users {
		if usr.ID != usrID {
			tmpUsers = append(tmpUsers, usr)
		}
	}
	s.Users = tmpUsers

	if len(s.Users) != 0 {
		s.Restart()
	} else {
		s.Stop()
	}
}
