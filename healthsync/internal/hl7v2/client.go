package hl7v2

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

// MLLP Frame Characters (Minimal Lower Layer Protocol)
const (
	MLLPStartBlock = 0x0B // Vertical Tab (VT)
	MLLPEndBlock   = 0x1C // File Separator (FS)
	MLLPCarriageR  = 0x0D // Carriage Return (CR)
)

// Client represents an HL7 v2.x MLLP client
type Client struct {
	mu           sync.RWMutex
	host         string
	port         int
	conn         net.Conn
	connected    bool
	useTLS       bool
	tlsConfig    *tls.Config
	timeout      time.Duration
	readTimeout  time.Duration
	writeTimeout time.Duration
	parser       *Parser
	messageID    int64
}

// ClientConfig holds client configuration
type ClientConfig struct {
	Host         string
	Port         int
	UseTLS       bool
	TLSConfig    *tls.Config
	Timeout      time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// NewClient creates a new HL7 v2.x client
func NewClient(config *ClientConfig) *Client {
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	readTimeout := config.ReadTimeout
	if readTimeout == 0 {
		readTimeout = 30 * time.Second
	}

	writeTimeout := config.WriteTimeout
	if writeTimeout == 0 {
		writeTimeout = 30 * time.Second
	}

	return &Client{
		host:         config.Host,
		port:         config.Port,
		useTLS:       config.UseTLS,
		tlsConfig:    config.TLSConfig,
		timeout:      timeout,
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
		parser:       NewParser(nil),
	}
}

// Connect establishes connection to the HL7 server
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	address := fmt.Sprintf("%s:%d", c.host, c.port)

	dialer := &net.Dialer{
		Timeout: c.timeout,
	}

	var conn net.Conn
	var err error

	if c.useTLS {
		tlsConfig := c.tlsConfig
		if tlsConfig == nil {
			tlsConfig = &tls.Config{
				MinVersion: tls.VersionTLS12,
			}
		}
		conn, err = tls.DialWithDialer(dialer, "tcp", address, tlsConfig)
	} else {
		conn, err = dialer.DialContext(ctx, "tcp", address)
	}

	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.conn = conn
	c.connected = true

	return nil
}

// Disconnect closes the connection
func (c *Client) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected || c.conn == nil {
		return nil
	}

	err := c.conn.Close()
	c.conn = nil
	c.connected = false

	return err
}

// IsConnected returns connection status
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// Send sends an HL7 message and returns the ACK response
func (c *Client) Send(ctx context.Context, msg *Message) (*Message, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	// Encode message
	encoded, err := c.encodeMessage(msg)
	if err != nil {
		return nil, fmt.Errorf("failed to encode message: %w", err)
	}

	// Wrap in MLLP frame
	frame := c.wrapMLLP(encoded)

	// Set write deadline
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
		return nil, fmt.Errorf("failed to set write deadline: %w", err)
	}

	// Send message
	if _, err := c.conn.Write(frame); err != nil {
		c.connected = false
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Set read deadline
	if err := c.conn.SetReadDeadline(time.Now().Add(c.readTimeout)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	// Read response
	response, err := c.readMLLPMessage()
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse response
	ack, err := c.parser.Parse(response)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ACK: %w", err)
	}

	return ack, nil
}

// SendRaw sends raw HL7 message bytes
func (c *Client) SendRaw(ctx context.Context, data []byte) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return nil, fmt.Errorf("not connected")
	}

	// Wrap in MLLP frame
	frame := c.wrapMLLP(data)

	// Set write deadline
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.writeTimeout)); err != nil {
		return nil, fmt.Errorf("failed to set write deadline: %w", err)
	}

	// Send message
	if _, err := c.conn.Write(frame); err != nil {
		c.connected = false
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Set read deadline
	if err := c.conn.SetReadDeadline(time.Now().Add(c.readTimeout)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	// Read response
	return c.readMLLPMessage()
}

// wrapMLLP wraps a message in MLLP frame
func (c *Client) wrapMLLP(data []byte) []byte {
	frame := make([]byte, 0, len(data)+3)
	frame = append(frame, MLLPStartBlock)
	frame = append(frame, data...)
	frame = append(frame, MLLPEndBlock, MLLPCarriageR)
	return frame
}

// readMLLPMessage reads an MLLP-framed message
func (c *Client) readMLLPMessage() ([]byte, error) {
	buf := make([]byte, 65536) // 64KB buffer
	var message []byte
	inMessage := false

	for {
		n, err := c.conn.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		for i := 0; i < n; i++ {
			b := buf[i]

			if b == MLLPStartBlock {
				inMessage = true
				message = message[:0]
				continue
			}

			if b == MLLPEndBlock {
				// Check for trailing CR
				if i+1 < n && buf[i+1] == MLLPCarriageR {
					i++
				}
				return message, nil
			}

			if inMessage {
				message = append(message, b)
			}
		}
	}

	if len(message) > 0 {
		return message, nil
	}

	return nil, fmt.Errorf("no complete message received")
}

// encodeMessage encodes an HL7 message to bytes
func (c *Client) encodeMessage(msg *Message) ([]byte, error) {
	var result []byte

	for _, seg := range msg.Segments {
		encoded, err := seg.Encode()
		if err != nil {
			return nil, fmt.Errorf("failed to encode segment %s: %w", seg.ID(), err)
		}
		result = append(result, []byte(encoded)...)
		result = append(result, '\r')
	}

	return result, nil
}

// NextMessageID generates the next message control ID
func (c *Client) NextMessageID() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.messageID++
	return fmt.Sprintf("%d", c.messageID)
}

// CreateADT creates an ADT message
func (c *Client) CreateADT(event TriggerEvent, patient *PID, visit *PV1) *Message {
	now := time.Now()
	msgID := c.NextMessageID()

	msh := &MSH{
		FieldSeparator:       DefaultFieldSeparator,
		EncodingCharacters:   "^~\\&",
		SendingApplication:   "SAVEGRESS",
		SendingFacility:      "HEALTHSYNC",
		ReceivingApplication: "",
		ReceivingFacility:    "",
		DateTime:             now,
		MessageType:          fmt.Sprintf("ADT%s%s", DefaultComponentSeparator, event),
		MessageControlID:     msgID,
		ProcessingID:         "P",
		VersionID:            "2.5.1",
	}

	evn := &EVN{
		EventTypeCode:    string(event),
		RecordedDateTime: now,
	}

	msg := &Message{
		Type:         MessageTypeADT,
		TriggerEvent: event,
		Version:      "2.5.1",
		ControlID:    msgID,
		Timestamp:    now,
		Segments:     []Segment{msh, evn, patient, visit},
	}

	return msg
}

// CreateORM creates an ORM (Order) message
func (c *Client) CreateORM(event TriggerEvent, patient *PID, order *ORC, request *OBR) *Message {
	now := time.Now()
	msgID := c.NextMessageID()

	msh := &MSH{
		FieldSeparator:       DefaultFieldSeparator,
		EncodingCharacters:   "^~\\&",
		SendingApplication:   "SAVEGRESS",
		SendingFacility:      "HEALTHSYNC",
		DateTime:             now,
		MessageType:          fmt.Sprintf("ORM%s%s", DefaultComponentSeparator, event),
		MessageControlID:     msgID,
		ProcessingID:         "P",
		VersionID:            "2.5.1",
	}

	msg := &Message{
		Type:         MessageTypeORM,
		TriggerEvent: event,
		Version:      "2.5.1",
		ControlID:    msgID,
		Timestamp:    now,
		Segments:     []Segment{msh, patient, order, request},
	}

	return msg
}

// CreateORU creates an ORU (Observation Result) message
func (c *Client) CreateORU(event TriggerEvent, patient *PID, request *OBR, observations []*OBX) *Message {
	now := time.Now()
	msgID := c.NextMessageID()

	msh := &MSH{
		FieldSeparator:       DefaultFieldSeparator,
		EncodingCharacters:   "^~\\&",
		SendingApplication:   "SAVEGRESS",
		SendingFacility:      "HEALTHSYNC",
		DateTime:             now,
		MessageType:          fmt.Sprintf("ORU%s%s", DefaultComponentSeparator, event),
		MessageControlID:     msgID,
		ProcessingID:         "P",
		VersionID:            "2.5.1",
	}

	segments := []Segment{msh, patient, request}
	for _, obx := range observations {
		segments = append(segments, obx)
	}

	msg := &Message{
		Type:         MessageTypeORU,
		TriggerEvent: event,
		Version:      "2.5.1",
		ControlID:    msgID,
		Timestamp:    now,
		Segments:     segments,
	}

	return msg
}

// CreateACK creates an ACK message
func (c *Client) CreateACK(originalMsg *Message, ackCode string, errorMessage string) *Message {
	now := time.Now()
	msgID := c.NextMessageID()

	msh := &MSH{
		FieldSeparator:       DefaultFieldSeparator,
		EncodingCharacters:   "^~\\&",
		SendingApplication:   "SAVEGRESS",
		SendingFacility:      "HEALTHSYNC",
		ReceivingApplication: originalMsg.SendingApp,
		ReceivingFacility:    originalMsg.SendingFac,
		DateTime:             now,
		MessageType:          "ACK",
		MessageControlID:     msgID,
		ProcessingID:         "P",
		VersionID:            "2.5.1",
	}

	msa := &MSA{
		AcknowledgmentCode:   ackCode,
		MessageControlID:     originalMsg.ControlID,
		TextMessage:          errorMessage,
	}

	msg := &Message{
		Type:      MessageTypeACK,
		Version:   "2.5.1",
		ControlID: msgID,
		Timestamp: now,
		Segments:  []Segment{msh, msa},
	}

	return msg
}

// MSA - Message Acknowledgment Segment
type MSA struct {
	AcknowledgmentCode   string // AA, AE, AR
	MessageControlID     string
	TextMessage          string
	ExpectedSequenceNum  string
	DelayedAckType       string
	ErrorCondition       CodedElement
}

func (m *MSA) ID() string { return "MSA" }

func (m *MSA) Encode() (string, error) {
	return fmt.Sprintf("MSA|%s|%s|%s", m.AcknowledgmentCode, m.MessageControlID, m.TextMessage), nil
}

func (m *MSA) Decode(data string) error {
	return nil
}

// Server represents an HL7 v2.x MLLP server
type Server struct {
	mu       sync.RWMutex
	listener net.Listener
	host     string
	port     int
	useTLS   bool
	tlsConfig *tls.Config
	running  bool
	parser   *Parser
	handler  MessageHandler
}

// MessageHandler handles incoming HL7 messages
type MessageHandler func(ctx context.Context, msg *Message) (*Message, error)

// ServerConfig holds server configuration
type ServerConfig struct {
	Host      string
	Port      int
	UseTLS    bool
	TLSConfig *tls.Config
	Handler   MessageHandler
}

// NewServer creates a new HL7 v2.x server
func NewServer(config *ServerConfig) *Server {
	return &Server{
		host:      config.Host,
		port:      config.Port,
		useTLS:    config.UseTLS,
		tlsConfig: config.TLSConfig,
		parser:    NewParser(nil),
		handler:   config.Handler,
	}
}

// Start starts the HL7 server
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()

	address := fmt.Sprintf("%s:%d", s.host, s.port)

	var listener net.Listener
	var err error

	if s.useTLS {
		listener, err = tls.Listen("tcp", address, s.tlsConfig)
	} else {
		listener, err = net.Listen("tcp", address)
	}

	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("failed to start server: %w", err)
	}

	s.listener = listener
	s.running = true
	s.mu.Unlock()

	go s.acceptConnections(ctx)

	return nil
}

// Stop stops the HL7 server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.running = false
	if s.listener != nil {
		return s.listener.Close()
	}

	return nil
}

// acceptConnections accepts incoming connections
func (s *Server) acceptConnections(ctx context.Context) {
	for {
		s.mu.RLock()
		running := s.running
		listener := s.listener
		s.mu.RUnlock()

		if !running {
			return
		}

		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		go s.handleConnection(ctx, conn)
	}
}

// handleConnection handles a client connection
func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 65536)
	var message []byte
	inMessage := false

	for {
		n, err := conn.Read(buf)
		if err != nil {
			return
		}

		for i := 0; i < n; i++ {
			b := buf[i]

			if b == MLLPStartBlock {
				inMessage = true
				message = message[:0]
				continue
			}

			if b == MLLPEndBlock {
				// Process message
				if len(message) > 0 {
					s.processMessage(ctx, conn, message)
				}
				inMessage = false
				message = message[:0]
				continue
			}

			if inMessage && b != MLLPCarriageR {
				message = append(message, b)
			}
		}
	}
}

// processMessage processes an incoming message
func (s *Server) processMessage(ctx context.Context, conn net.Conn, data []byte) {
	msg, err := s.parser.Parse(data)
	if err != nil {
		// Send NAK
		return
	}

	var response *Message
	if s.handler != nil {
		response, err = s.handler(ctx, msg)
		if err != nil {
			// Create error ACK
			client := &Client{}
			response = client.CreateACK(msg, "AE", err.Error())
		}
	} else {
		// Create default ACK
		client := &Client{}
		response = client.CreateACK(msg, "AA", "")
	}

	if response != nil {
		client := &Client{}
		encoded, err := client.encodeMessage(response)
		if err != nil {
			return
		}

		frame := client.wrapMLLP(encoded)
		conn.Write(frame)
	}
}
