package input

// Controller represents a NES controller
type Controller struct {
	// Button states
	buttons   uint8
	strobe    bool
	index     uint8
	
	// Button mapping
	ButtonA      bool
	ButtonB      bool
	ButtonSelect bool
	ButtonStart  bool
	ButtonUp     bool
	ButtonDown   bool
	ButtonLeft   bool
	ButtonRight  bool
}

// Button constants
const (
	ButtonMaskA      = 1 << 0
	ButtonMaskB      = 1 << 1
	ButtonMaskSelect = 1 << 2
	ButtonMaskStart  = 1 << 3
	ButtonMaskUp     = 1 << 4
	ButtonMaskDown   = 1 << 5
	ButtonMaskLeft   = 1 << 6
	ButtonMaskRight  = 1 << 7
)

// New creates a new Controller instance
func New() *Controller {
	return &Controller{}
}

// SetButton sets the state of a button
func (c *Controller) SetButton(controller int, button int, pressed bool) {
	// For now, only support controller 0
	if controller != 0 {
		return
	}
	
	buttonMask := uint8(1 << button)
	
	if pressed {
		c.buttons |= buttonMask
	} else {
		c.buttons &^= buttonMask
	}
	
	// Update individual button states for easier access
	c.ButtonA = c.buttons&ButtonMaskA != 0
	c.ButtonB = c.buttons&ButtonMaskB != 0
	c.ButtonSelect = c.buttons&ButtonMaskSelect != 0
	c.ButtonStart = c.buttons&ButtonMaskStart != 0
	c.ButtonUp = c.buttons&ButtonMaskUp != 0
	c.ButtonDown = c.buttons&ButtonMaskDown != 0
	c.ButtonLeft = c.buttons&ButtonMaskLeft != 0
	c.ButtonRight = c.buttons&ButtonMaskRight != 0
}

// Read reads the controller state
func (c *Controller) Read() uint8 {
	if c.index > 7 {
		return 1
	}
	
	result := (c.buttons >> c.index) & 1
	
	if !c.strobe {
		c.index++
	}
	
	return result
}

// Write writes to the controller (strobe)
func (c *Controller) Write(value uint8) {
	c.strobe = value&1 != 0
	if c.strobe {
		c.index = 0
	}
}

// GetButtons returns the current button state
func (c *Controller) GetButtons() uint8 {
	return c.buttons
}

// IsPressed checks if a specific button is pressed
func (c *Controller) IsPressed(button uint8) bool {
	return c.buttons&button != 0
}