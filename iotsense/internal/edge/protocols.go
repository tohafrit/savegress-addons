package edge

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// ProtocolHandler interface for protocol adapters
type ProtocolHandler interface {
	Connect(ctx context.Context, config map[string]interface{}) error
	Disconnect() error
	Subscribe(topic string, handler func(data []byte)) error
	Publish(topic string, data []byte) error
	Read(address string) (interface{}, error)
	Write(address string, value interface{}) error
	IsConnected() bool
}

// MQTTHandler handles MQTT protocol
type MQTTHandler struct {
	mu         sync.RWMutex
	broker     string
	port       int
	clientID   string
	username   string
	password   string
	useTLS     bool
	connected  bool
	subscriptions map[string]func(data []byte)
}

// NewMQTTHandler creates a new MQTT handler
func NewMQTTHandler() *MQTTHandler {
	return &MQTTHandler{
		subscriptions: make(map[string]func(data []byte)),
	}
}

// Connect connects to the MQTT broker
func (h *MQTTHandler) Connect(ctx context.Context, config map[string]interface{}) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if broker, ok := config["broker"].(string); ok {
		h.broker = broker
	}
	if port, ok := config["port"].(int); ok {
		h.port = port
	}
	if clientID, ok := config["client_id"].(string); ok {
		h.clientID = clientID
	}
	if username, ok := config["username"].(string); ok {
		h.username = username
	}
	if password, ok := config["password"].(string); ok {
		h.password = password
	}
	if useTLS, ok := config["use_tls"].(bool); ok {
		h.useTLS = useTLS
	}

	// In a real implementation, this would connect to the MQTT broker
	// using a library like paho-mqtt

	h.connected = true
	return nil
}

// Disconnect disconnects from the MQTT broker
func (h *MQTTHandler) Disconnect() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.connected = false
	h.subscriptions = make(map[string]func(data []byte))

	return nil
}

// Subscribe subscribes to a topic
func (h *MQTTHandler) Subscribe(topic string, handler func(data []byte)) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.connected {
		return fmt.Errorf("not connected")
	}

	h.subscriptions[topic] = handler

	// In a real implementation, this would subscribe to the MQTT topic

	return nil
}

// Publish publishes a message to a topic
func (h *MQTTHandler) Publish(topic string, data []byte) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.connected {
		return fmt.Errorf("not connected")
	}

	// In a real implementation, this would publish to the MQTT broker

	return nil
}

// Read is not typically used for MQTT
func (h *MQTTHandler) Read(address string) (interface{}, error) {
	return nil, fmt.Errorf("read not supported for MQTT")
}

// Write is not typically used for MQTT
func (h *MQTTHandler) Write(address string, value interface{}) error {
	return fmt.Errorf("write not supported for MQTT")
}

// IsConnected returns connection status
func (h *MQTTHandler) IsConnected() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.connected
}

// ModbusHandler handles Modbus protocol
type ModbusHandler struct {
	mu        sync.RWMutex
	address   string
	port      int
	slaveID   byte
	connected bool
	timeout   time.Duration
}

// NewModbusHandler creates a new Modbus handler
func NewModbusHandler() *ModbusHandler {
	return &ModbusHandler{
		timeout: 5 * time.Second,
	}
}

// Connect connects to a Modbus device
func (h *ModbusHandler) Connect(ctx context.Context, config map[string]interface{}) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if address, ok := config["address"].(string); ok {
		h.address = address
	}
	if port, ok := config["port"].(int); ok {
		h.port = port
	}
	if slaveID, ok := config["slave_id"].(int); ok {
		h.slaveID = byte(slaveID)
	}
	if timeout, ok := config["timeout"].(time.Duration); ok {
		h.timeout = timeout
	}

	// In a real implementation, this would connect to the Modbus device

	h.connected = true
	return nil
}

// Disconnect disconnects from the Modbus device
func (h *ModbusHandler) Disconnect() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.connected = false
	return nil
}

// Subscribe is not supported for Modbus
func (h *ModbusHandler) Subscribe(topic string, handler func(data []byte)) error {
	return fmt.Errorf("subscribe not supported for Modbus")
}

// Publish is not supported for Modbus
func (h *ModbusHandler) Publish(topic string, data []byte) error {
	return fmt.Errorf("publish not supported for Modbus")
}

// Read reads from a Modbus register
func (h *ModbusHandler) Read(address string) (interface{}, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.connected {
		return nil, fmt.Errorf("not connected")
	}

	// Parse address format: "holding:40001" or "input:30001"
	var registerType string
	var registerAddr int
	_, err := fmt.Sscanf(address, "%s:%d", &registerType, &registerAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid address format: %s", address)
	}

	// In a real implementation, this would read from the Modbus device

	return uint16(0), nil
}

// Write writes to a Modbus register
func (h *ModbusHandler) Write(address string, value interface{}) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.connected {
		return fmt.Errorf("not connected")
	}

	// In a real implementation, this would write to the Modbus device

	return nil
}

// IsConnected returns connection status
func (h *ModbusHandler) IsConnected() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.connected
}

// OPCUAHandler handles OPC-UA protocol
type OPCUAHandler struct {
	mu          sync.RWMutex
	endpointURL string
	connected   bool
	security    string
	username    string
	password    string
	subscriptions map[string]func(data []byte)
}

// NewOPCUAHandler creates a new OPC-UA handler
func NewOPCUAHandler() *OPCUAHandler {
	return &OPCUAHandler{
		subscriptions: make(map[string]func(data []byte)),
	}
}

// Connect connects to an OPC-UA server
func (h *OPCUAHandler) Connect(ctx context.Context, config map[string]interface{}) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if endpoint, ok := config["endpoint"].(string); ok {
		h.endpointURL = endpoint
	}
	if security, ok := config["security"].(string); ok {
		h.security = security
	}
	if username, ok := config["username"].(string); ok {
		h.username = username
	}
	if password, ok := config["password"].(string); ok {
		h.password = password
	}

	// In a real implementation, this would connect to the OPC-UA server

	h.connected = true
	return nil
}

// Disconnect disconnects from the OPC-UA server
func (h *OPCUAHandler) Disconnect() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.connected = false
	h.subscriptions = make(map[string]func(data []byte))

	return nil
}

// Subscribe subscribes to an OPC-UA node
func (h *OPCUAHandler) Subscribe(nodeID string, handler func(data []byte)) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.connected {
		return fmt.Errorf("not connected")
	}

	h.subscriptions[nodeID] = handler

	// In a real implementation, this would create a subscription to the node

	return nil
}

// Publish is not typically used for OPC-UA
func (h *OPCUAHandler) Publish(topic string, data []byte) error {
	return fmt.Errorf("publish not supported for OPC-UA")
}

// Read reads from an OPC-UA node
func (h *OPCUAHandler) Read(nodeID string) (interface{}, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.connected {
		return nil, fmt.Errorf("not connected")
	}

	// In a real implementation, this would read from the OPC-UA node

	return nil, nil
}

// Write writes to an OPC-UA node
func (h *OPCUAHandler) Write(nodeID string, value interface{}) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.connected {
		return fmt.Errorf("not connected")
	}

	// In a real implementation, this would write to the OPC-UA node

	return nil
}

// IsConnected returns connection status
func (h *OPCUAHandler) IsConnected() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.connected
}

// BACnetHandler handles BACnet protocol (for building automation)
type BACnetHandler struct {
	mu        sync.RWMutex
	deviceID  int
	port      int
	connected bool
}

// NewBACnetHandler creates a new BACnet handler
func NewBACnetHandler() *BACnetHandler {
	return &BACnetHandler{
		port: 47808, // Default BACnet port
	}
}

// Connect connects to a BACnet network
func (h *BACnetHandler) Connect(ctx context.Context, config map[string]interface{}) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if deviceID, ok := config["device_id"].(int); ok {
		h.deviceID = deviceID
	}
	if port, ok := config["port"].(int); ok {
		h.port = port
	}

	// In a real implementation, this would initialize BACnet communication

	h.connected = true
	return nil
}

// Disconnect disconnects from the BACnet network
func (h *BACnetHandler) Disconnect() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.connected = false
	return nil
}

// Subscribe subscribes to a BACnet object
func (h *BACnetHandler) Subscribe(objectID string, handler func(data []byte)) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.connected {
		return fmt.Errorf("not connected")
	}

	// In a real implementation, this would create a COV subscription

	return nil
}

// Publish is not supported for BACnet
func (h *BACnetHandler) Publish(topic string, data []byte) error {
	return fmt.Errorf("publish not supported for BACnet")
}

// Read reads a BACnet property
func (h *BACnetHandler) Read(objectProperty string) (interface{}, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.connected {
		return nil, fmt.Errorf("not connected")
	}

	// In a real implementation, this would read the BACnet property

	return nil, nil
}

// Write writes a BACnet property
func (h *BACnetHandler) Write(objectProperty string, value interface{}) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.connected {
		return fmt.Errorf("not connected")
	}

	// In a real implementation, this would write the BACnet property

	return nil
}

// IsConnected returns connection status
func (h *BACnetHandler) IsConnected() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.connected
}

// ProtocolRegistry manages protocol handlers
type ProtocolRegistry struct {
	mu       sync.RWMutex
	handlers map[string]func() ProtocolHandler
}

// NewProtocolRegistry creates a new protocol registry
func NewProtocolRegistry() *ProtocolRegistry {
	registry := &ProtocolRegistry{
		handlers: make(map[string]func() ProtocolHandler),
	}

	// Register default handlers
	registry.Register("mqtt", func() ProtocolHandler { return NewMQTTHandler() })
	registry.Register("modbus", func() ProtocolHandler { return NewModbusHandler() })
	registry.Register("opcua", func() ProtocolHandler { return NewOPCUAHandler() })
	registry.Register("bacnet", func() ProtocolHandler { return NewBACnetHandler() })
	registry.Register("amqp", func() ProtocolHandler { return NewAMQPHandler() })
	registry.Register("coap", func() ProtocolHandler { return NewCoAPHandler() })

	return registry
}

// Register registers a protocol handler factory
func (r *ProtocolRegistry) Register(protocol string, factory func() ProtocolHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[protocol] = factory
}

// Create creates a handler for the given protocol
func (r *ProtocolRegistry) Create(protocol string) (ProtocolHandler, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, ok := r.handlers[protocol]
	if !ok {
		return nil, fmt.Errorf("unknown protocol: %s", protocol)
	}

	return factory(), nil
}

// ListProtocols returns supported protocols
func (r *ProtocolRegistry) ListProtocols() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	protocols := make([]string, 0, len(r.handlers))
	for protocol := range r.handlers {
		protocols = append(protocols, protocol)
	}

	return protocols
}

// AMQPHandler handles AMQP (Advanced Message Queuing Protocol) for IoT
type AMQPHandler struct {
	mu            sync.RWMutex
	host          string
	port          int
	vhost         string
	username      string
	password      string
	useTLS        bool
	connected     bool
	subscriptions map[string]func(data []byte)
	exchange      string
	exchangeType  string // direct, fanout, topic, headers
	durable       bool
	autoDelete    bool
}

// NewAMQPHandler creates a new AMQP handler
func NewAMQPHandler() *AMQPHandler {
	return &AMQPHandler{
		port:          5672,
		vhost:         "/",
		subscriptions: make(map[string]func(data []byte)),
		exchangeType:  "topic",
		durable:       true,
	}
}

// Connect connects to the AMQP broker
func (h *AMQPHandler) Connect(ctx context.Context, config map[string]interface{}) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if host, ok := config["host"].(string); ok {
		h.host = host
	}
	if port, ok := config["port"].(int); ok {
		h.port = port
	}
	if vhost, ok := config["vhost"].(string); ok {
		h.vhost = vhost
	}
	if username, ok := config["username"].(string); ok {
		h.username = username
	}
	if password, ok := config["password"].(string); ok {
		h.password = password
	}
	if useTLS, ok := config["use_tls"].(bool); ok {
		h.useTLS = useTLS
	}
	if exchange, ok := config["exchange"].(string); ok {
		h.exchange = exchange
	}
	if exchangeType, ok := config["exchange_type"].(string); ok {
		h.exchangeType = exchangeType
	}
	if durable, ok := config["durable"].(bool); ok {
		h.durable = durable
	}
	if autoDelete, ok := config["auto_delete"].(bool); ok {
		h.autoDelete = autoDelete
	}

	// In a real implementation, this would connect using streadway/amqp:
	// conn, err := amqp.Dial(fmt.Sprintf("amqp://%s:%s@%s:%d/%s",
	//     h.username, h.password, h.host, h.port, h.vhost))
	// ch, err := conn.Channel()
	// err = ch.ExchangeDeclare(h.exchange, h.exchangeType, h.durable, h.autoDelete, false, false, nil)

	h.connected = true
	return nil
}

// Disconnect disconnects from the AMQP broker
func (h *AMQPHandler) Disconnect() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.connected = false
	h.subscriptions = make(map[string]func(data []byte))

	return nil
}

// Subscribe subscribes to an AMQP queue with routing key
func (h *AMQPHandler) Subscribe(routingKey string, handler func(data []byte)) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.connected {
		return fmt.Errorf("not connected")
	}

	h.subscriptions[routingKey] = handler

	// In a real implementation:
	// q, err := ch.QueueDeclare("", false, true, true, false, nil)
	// err = ch.QueueBind(q.Name, routingKey, h.exchange, false, nil)
	// msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	// go func() {
	//     for d := range msgs {
	//         handler(d.Body)
	//     }
	// }()

	return nil
}

// Publish publishes a message to the AMQP exchange
func (h *AMQPHandler) Publish(routingKey string, data []byte) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.connected {
		return fmt.Errorf("not connected")
	}

	// In a real implementation:
	// err = ch.Publish(h.exchange, routingKey, false, false, amqp.Publishing{
	//     ContentType: "application/json",
	//     Body:        data,
	// })

	return nil
}

// Read is not typically used for AMQP (async)
func (h *AMQPHandler) Read(address string) (interface{}, error) {
	return nil, fmt.Errorf("read not supported for AMQP, use Subscribe")
}

// Write is an alias for Publish
func (h *AMQPHandler) Write(address string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal: %w", err)
	}
	return h.Publish(address, data)
}

// IsConnected returns connection status
func (h *AMQPHandler) IsConnected() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.connected
}

// AMQPMessage represents an AMQP message
type AMQPMessage struct {
	RoutingKey  string
	Exchange    string
	ContentType string
	Body        []byte
	Headers     map[string]interface{}
	Timestamp   time.Time
	MessageID   string
	CorrelationID string
	ReplyTo     string
	Priority    uint8
	DeliveryMode uint8 // 1=transient, 2=persistent
}

// PublishWithOptions publishes with full AMQP message options
func (h *AMQPHandler) PublishWithOptions(msg *AMQPMessage) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.connected {
		return fmt.Errorf("not connected")
	}

	// In a real implementation, this would use all message properties

	return nil
}

// CoAPHandler handles CoAP (Constrained Application Protocol) for IoT
type CoAPHandler struct {
	mu            sync.RWMutex
	host          string
	port          int
	connected     bool
	useDTLS       bool
	pskIdentity   string
	pskKey        []byte
	observations  map[string]func(data []byte)
	timeout       time.Duration
}

// NewCoAPHandler creates a new CoAP handler
func NewCoAPHandler() *CoAPHandler {
	return &CoAPHandler{
		port:         5683, // Default CoAP port (5684 for DTLS)
		observations: make(map[string]func(data []byte)),
		timeout:      5 * time.Second,
	}
}

// Connect initializes the CoAP client
func (h *CoAPHandler) Connect(ctx context.Context, config map[string]interface{}) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if host, ok := config["host"].(string); ok {
		h.host = host
	}
	if port, ok := config["port"].(int); ok {
		h.port = port
	}
	if useDTLS, ok := config["use_dtls"].(bool); ok {
		h.useDTLS = useDTLS
		if h.useDTLS && h.port == 5683 {
			h.port = 5684 // Default DTLS port
		}
	}
	if pskIdentity, ok := config["psk_identity"].(string); ok {
		h.pskIdentity = pskIdentity
	}
	if pskKey, ok := config["psk_key"].([]byte); ok {
		h.pskKey = pskKey
	}
	if timeout, ok := config["timeout"].(time.Duration); ok {
		h.timeout = timeout
	}

	// In a real implementation, this would initialize the CoAP client:
	// client := coap.Client{}
	// if h.useDTLS {
	//     client = coap.DTLSClient(pskIdentity, pskKey)
	// }

	h.connected = true
	return nil
}

// Disconnect closes CoAP connections
func (h *CoAPHandler) Disconnect() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.connected = false
	h.observations = make(map[string]func(data []byte))

	return nil
}

// Subscribe creates a CoAP Observe subscription
func (h *CoAPHandler) Subscribe(path string, handler func(data []byte)) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.connected {
		return fmt.Errorf("not connected")
	}

	h.observations[path] = handler

	// In a real implementation, this would use CoAP Observe (RFC 7641):
	// req := coap.Message{
	//     Type:      coap.Confirmable,
	//     Code:      coap.GET,
	//     MessageID: generateMessageID(),
	//     Observe:   0, // Register observation
	// }
	// req.SetPath(path)
	// conn.WriteMsg(req)
	// go func() {
	//     for {
	//         resp, err := conn.ReadMsg()
	//         if err != nil { break }
	//         handler(resp.Payload)
	//     }
	// }()

	return nil
}

// Publish sends a CoAP POST request
func (h *CoAPHandler) Publish(path string, data []byte) error {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.connected {
		return fmt.Errorf("not connected")
	}

	// In a real implementation:
	// req := coap.Message{
	//     Type:    coap.Confirmable,
	//     Code:    coap.POST,
	//     Payload: data,
	// }
	// req.SetPath(path)
	// resp, err := conn.Exchange(req)

	return nil
}

// Read performs a CoAP GET request
func (h *CoAPHandler) Read(path string) (interface{}, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.connected {
		return nil, fmt.Errorf("not connected")
	}

	// In a real implementation:
	// req := coap.Message{
	//     Type: coap.Confirmable,
	//     Code: coap.GET,
	// }
	// req.SetPath(path)
	// resp, err := conn.Exchange(req)
	// return resp.Payload, nil

	return nil, nil
}

// Write performs a CoAP PUT request
func (h *CoAPHandler) Write(path string, value interface{}) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.connected {
		return fmt.Errorf("not connected")
	}

	// In a real implementation:
	// data, err := json.Marshal(value)
	// req := coap.Message{
	//     Type:    coap.Confirmable,
	//     Code:    coap.PUT,
	//     Payload: data,
	// }
	// req.SetPath(path)
	// resp, err := conn.Exchange(req)

	return nil
}

// IsConnected returns connection status
func (h *CoAPHandler) IsConnected() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.connected
}

// CoAPMethod represents CoAP request methods
type CoAPMethod uint8

const (
	CoAPGET    CoAPMethod = 1
	CoAPPOST   CoAPMethod = 2
	CoAPPUT    CoAPMethod = 3
	CoAPDELETE CoAPMethod = 4
)

// CoAPContentFormat represents CoAP content formats
type CoAPContentFormat uint16

const (
	CoAPTextPlain         CoAPContentFormat = 0
	CoAPLinkFormat        CoAPContentFormat = 40
	CoAPXML               CoAPContentFormat = 41
	CoAPOctetStream       CoAPContentFormat = 42
	CoAPEXI               CoAPContentFormat = 47
	CoAPJSON              CoAPContentFormat = 50
	CoAPCBOR              CoAPContentFormat = 60
	CoAPSenMLJSON         CoAPContentFormat = 110
	CoAPSenMLCBOR         CoAPContentFormat = 112
	CoAPLwM2MTLV          CoAPContentFormat = 11542
	CoAPLwM2MJSON         CoAPContentFormat = 11543
)

// CoAPMessage represents a CoAP message
type CoAPMessage struct {
	Method        CoAPMethod
	Path          string
	Payload       []byte
	ContentFormat CoAPContentFormat
	Token         []byte
	MessageID     uint16
	Observe       *uint32 // nil=no observe, 0=register, 1=deregister
	Options       map[string]interface{}
}

// Request performs a CoAP request with full message options
func (h *CoAPHandler) Request(msg *CoAPMessage) ([]byte, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.connected {
		return nil, fmt.Errorf("not connected")
	}

	// In a real implementation, this would construct and send the CoAP message

	return nil, nil
}

// Discover performs CoAP resource discovery (/.well-known/core)
func (h *CoAPHandler) Discover() ([]CoAPResource, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.connected {
		return nil, fmt.Errorf("not connected")
	}

	// In a real implementation:
	// resp, err := h.Read("/.well-known/core")
	// Parse link-format response

	return nil, nil
}

// CoAPResource represents a discoverable CoAP resource
type CoAPResource struct {
	Path          string
	ResourceType  string   // rt=
	InterfaceDesc string   // if=
	ContentFormat []uint16 // ct=
	Size          uint32   // sz=
	Title         string   // title=
	Observable    bool     // obs
}

// CancelObserve cancels an observation
func (h *CoAPHandler) CancelObserve(path string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.connected {
		return fmt.Errorf("not connected")
	}

	delete(h.observations, path)

	// In a real implementation:
	// req := coap.Message{
	//     Type:    coap.Reset,
	//     Code:    coap.GET,
	//     Observe: 1, // Deregister
	// }
	// req.SetPath(path)

	return nil
}

// DataConverter converts data between protocols
type DataConverter struct{}

// NewDataConverter creates a new data converter
func NewDataConverter() *DataConverter {
	return &DataConverter{}
}

// ToJSON converts data to JSON
func (c *DataConverter) ToJSON(data interface{}) ([]byte, error) {
	return json.Marshal(data)
}

// FromJSON converts JSON to data
func (c *DataConverter) FromJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// ToModbus converts a value to Modbus register format
func (c *DataConverter) ToModbus(value interface{}) ([]uint16, error) {
	switch v := value.(type) {
	case int16:
		return []uint16{uint16(v)}, nil
	case uint16:
		return []uint16{v}, nil
	case int32:
		return []uint16{
			uint16(v >> 16),
			uint16(v & 0xFFFF),
		}, nil
	case uint32:
		return []uint16{
			uint16(v >> 16),
			uint16(v & 0xFFFF),
		}, nil
	case float32:
		bits := binary.BigEndian.Uint32([]byte{0, 0, 0, 0})
		return []uint16{
			uint16(bits >> 16),
			uint16(bits & 0xFFFF),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported type for Modbus conversion")
	}
}

// FromModbus converts Modbus registers to a value
func (c *DataConverter) FromModbus(registers []uint16, dataType string) (interface{}, error) {
	switch dataType {
	case "int16":
		if len(registers) < 1 {
			return nil, fmt.Errorf("insufficient registers")
		}
		return int16(registers[0]), nil
	case "uint16":
		if len(registers) < 1 {
			return nil, fmt.Errorf("insufficient registers")
		}
		return registers[0], nil
	case "int32":
		if len(registers) < 2 {
			return nil, fmt.Errorf("insufficient registers")
		}
		return int32(registers[0])<<16 | int32(registers[1]), nil
	case "uint32":
		if len(registers) < 2 {
			return nil, fmt.Errorf("insufficient registers")
		}
		return uint32(registers[0])<<16 | uint32(registers[1]), nil
	default:
		return nil, fmt.Errorf("unsupported data type: %s", dataType)
	}
}
