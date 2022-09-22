// copied from gobot
package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/pkg/errors"

	"tinygo.org/x/bluetooth"
)

//var currentDevice *blelib.Device
var currentAdapter *bluetooth.Adapter
var bleMutex sync.Mutex

// ClientAdaptor represents a Client Connection to a BLE Peripheral
type ClientAdaptor struct {
	name        string
	address     string
	AdapterName string

	addr            bluetooth.Address
	adpt            *bluetooth.Adapter
	device          *bluetooth.Device
	characteristics map[string]bluetooth.DeviceCharacteristic

	connected        bool
	ready            chan struct{}
	withoutResponses bool
}

// NewClientAdaptor returns a new ClientAdaptor given an address
func NewClientAdaptor(name string) *ClientAdaptor {
	return &ClientAdaptor{
		name:             name,
		address:          "",
		AdapterName:      "default",
		connected:        false,
		withoutResponses: false,
		characteristics:  make(map[string]bluetooth.DeviceCharacteristic),
	}
}

// Name returns the name for the adaptor
func (b *ClientAdaptor) Name() string { return b.name }

// SetName sets the name for the adaptor
func (b *ClientAdaptor) SetName(n string) { b.name = n }

// Address returns the Bluetooth LE address for the adaptor
func (b *ClientAdaptor) Address() string { return b.address }

// WithoutResponses sets if the adaptor should expect responses after
// writing characteristics for this device
func (b *ClientAdaptor) WithoutResponses(use bool) { b.withoutResponses = use }

// Connect initiates a connection to the BLE peripheral. Returns true on successful connection.
func (b *ClientAdaptor) Connect() (err error) {
	bleMutex.Lock()
	defer bleMutex.Unlock()

	// enable adaptor
	b.adpt, err = getBLEAdapter(b.AdapterName)
	if err != nil {
		return errors.Wrap(err, "can't enable adapter "+b.AdapterName)
	}

	// scan for the address
	ch := make(chan bluetooth.ScanResult, 1)
	err = b.adpt.Scan(func(adapter *bluetooth.Adapter, result bluetooth.ScanResult) {
		if result.LocalName() == b.Name() {
			fmt.Println("found device", result.LocalName(), result.Address.String())
			b.adpt.StopScan()
			b.address = result.Address.String()
			ch <- result
		} else {
			fmt.Println("skipping device", result.LocalName(), result.Address.String())
		}
	})

	if err != nil {
		return err
	}

	// handle address
	b.addr.Set(b.Address())

	// wait to connect to peripheral device
	select {
	case result := <-ch:
		b.device, err = b.adpt.Connect(result.Address, bluetooth.ConnectionParams{})
		if err != nil {
			return err
		}
	}

	// get all services/characteristics
	srvcs, err := b.device.DiscoverServices(nil)
	for _, srvc := range srvcs {
		chars, err := srvc.DiscoverCharacteristics(nil)
		if err != nil {
			log.Println(err)
			continue
		}
		for _, char := range chars {
			b.characteristics[char.UUID().String()] = char
		}
	}

	b.connected = true
	return
}

// Reconnect attempts to reconnect to the BLE peripheral. If it has an active connection
// it will first close that connection and then establish a new connection.
// Returns true on Successful reconnection
func (b *ClientAdaptor) Reconnect() (err error) {
	if b.connected {
		b.Disconnect()
	}
	return b.Connect()
}

// Disconnect terminates the connection to the BLE peripheral. Returns true on successful disconnect.
func (b *ClientAdaptor) Disconnect() (err error) {
	err = b.device.Disconnect()
	time.Sleep(500 * time.Millisecond)
	return
}

// Finalize finalizes the BLEAdaptor
func (b *ClientAdaptor) Finalize() (err error) {
	return b.Disconnect()
}

// ReadCharacteristic returns bytes from the BLE device for the
// requested characteristic uuid
func (b *ClientAdaptor) ReadCharacteristic(cUUID string) (data []byte, err error) {
	if !b.connected {
		log.Fatalf("Cannot read from BLE device until connected")
		return
	}

	cUUID = convertUUID(cUUID)

	if char, ok := b.characteristics[cUUID]; ok {
		buf := make([]byte, 255)
		n, err := char.Read(buf)
		if err != nil {
			return nil, err
		}
		return buf[:n], nil
	}

	return nil, fmt.Errorf("Unknown characteristic: %s", cUUID)
}

// WriteCharacteristic writes bytes to the BLE device for the
// requested service and characteristic
func (b *ClientAdaptor) WriteCharacteristic(cUUID string, data []byte) (err error) {
	if !b.connected {
		log.Println("Cannot write to BLE device until connected")
		return
	}

	cUUID = convertUUID(cUUID)

	if char, ok := b.characteristics[cUUID]; ok {
		_, err := char.WriteWithoutResponse(data)
		if err != nil {
			return err
		}
		return nil
	}

	return fmt.Errorf("Unknown characteristic: %s", cUUID)
}

// Subscribe subscribes to notifications from the BLE device for the
// requested service and characteristic
func (b *ClientAdaptor) Subscribe(cUUID string, f func([]byte, error)) (err error) {
	if !b.connected {
		log.Fatalf("Cannot subscribe to BLE device until connected")
		return
	}

	cUUID = convertUUID(cUUID)

	if char, ok := b.characteristics[cUUID]; ok {
		fn := func(d []byte) {
			f(d, nil)
		}
		err = char.EnableNotifications(fn)
		return
	}

	return fmt.Errorf("Unknown characteristic: %s", cUUID)
}

// getBLEDevice is singleton for bluetooth adapter connection
func getBLEAdapter(impl string) (*bluetooth.Adapter, error) {
	if currentAdapter != nil {
		return currentAdapter, nil
	}

	currentAdapter = bluetooth.DefaultAdapter
	err := currentAdapter.Enable()
	if err != nil {
		return nil, errors.Wrap(err, "can't get device")
	}

	return currentAdapter, nil
}

func convertUUID(cUUID string) string {
	switch len(cUUID) {
	case 4:
		// 2a270000-0000-0000-0000-000000000000
		// convert "22bb"
		// to "22bb0000-0000-0000-0000-000000000000"
		return fmt.Sprintf("%s0000-0000-0000-0000-000000000000", cUUID)
	case 32:
		// convert "22bb746f2bbd75542d6f726568705327"
		// to "22bb746f-2bbd-7554-2d6f-726568705327"
		return fmt.Sprintf("%s-%s-%s-%s-%s", cUUID[:8], cUUID[8:12], cUUID[12:16], cUUID[16:20],
			cUUID[20:32])
	}

	return cUUID
}
