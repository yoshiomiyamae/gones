package gui

import (
	"github.com/veandco/go-sdl2/sdl"
	"github.com/yoshiomiyamaegones/pkg/logger"
	"github.com/yoshiomiyamaegones/pkg/nes"
)

// InputManager handles all input devices (keyboard, joystick, gamepad)
type InputManager struct {
	nes             *nes.NES
	joysticks       []*sdl.Joystick
	gameControllers []*sdl.GameController
}

// NewInputManager creates a new input manager
func NewInputManager(nesSystem *nes.NES) *InputManager {
	return &InputManager{
		nes:             nesSystem,
		joysticks:       make([]*sdl.Joystick, 0, 2),
		gameControllers: make([]*sdl.GameController, 0, 2),
	}
}

// Initialize prepares the input manager. Devices are opened lazily via the
// SDL hot-plug events (JOYDEVICEADDED / CONTROLLERDEVICEADDED), which are
// emitted both for devices present at startup and for any plugged in later.
// This means a controller connected mid-session is picked up automatically.
func (im *InputManager) Initialize() {
	logger.LogInfo("Input ready (controllers detected via hot-plug events). Use keyboard if none connected.")
}

// maxControllers is the limit (NES has two controller ports).
const maxControllers = 2

// gameControllerSlot returns the slice index of an opened SDL GameController
// matching the given instance ID, or -1 if not found. Used to route
// per-device events (button/axis/remove) to the right NES player slot,
// since slot order can shift when devices are unplugged mid-session.
func (im *InputManager) gameControllerSlot(id sdl.JoystickID) int {
	for i, c := range im.gameControllers {
		if c == nil {
			continue
		}
		if joy := c.Joystick(); joy != nil && joy.InstanceID() == id {
			return i
		}
	}
	return -1
}

// joystickSlot is the raw-joystick counterpart of gameControllerSlot.
func (im *InputManager) joystickSlot(id sdl.JoystickID) int {
	for i, j := range im.joysticks {
		if j == nil {
			continue
		}
		if j.InstanceID() == id {
			return i
		}
	}
	return -1
}

// Cleanup releases all input devices
func (im *InputManager) Cleanup() {
	// Close game controllers
	for _, controller := range im.gameControllers {
		if controller != nil {
			controller.Close()
		}
	}

	// Close joysticks
	for _, joystick := range im.joysticks {
		if joystick != nil {
			joystick.Close()
		}
	}
}

// HandleEvent processes input events
func (im *InputManager) HandleEvent(event sdl.Event) bool {
	switch e := event.(type) {
	case *sdl.KeyboardEvent:
		im.handleKeyboard(e)
		return true
	case *sdl.ControllerButtonEvent:
		im.handleControllerButton(e)
		return true
	case *sdl.ControllerAxisEvent:
		im.handleControllerAxis(e)
		return true
	case *sdl.ControllerDeviceEvent:
		im.handleControllerDevice(e)
		return true
	case *sdl.JoyButtonEvent:
		im.handleJoyButton(e)
		return true
	case *sdl.JoyAxisEvent:
		im.handleJoyAxis(e)
		return true
	case *sdl.JoyHatEvent:
		im.handleJoyHat(e)
		return true
	case *sdl.JoyDeviceAddedEvent:
		im.handleJoyDeviceAdded(e)
		return true
	case *sdl.JoyDeviceRemovedEvent:
		im.handleJoyDeviceRemoved(e)
		return true
	}
	return false
}

// handleControllerDevice opens/closes SDL GameControllers as they plug in or
// out. ADDED's Which is the device index; REMOVED's Which is the instance ID.
func (im *InputManager) handleControllerDevice(event *sdl.ControllerDeviceEvent) {
	switch event.Type {
	case sdl.CONTROLLERDEVICEADDED:
		if len(im.gameControllers) >= maxControllers {
			logger.LogInfo("Ignoring extra GameController (NES supports at most %d)", maxControllers)
			return
		}
		controller := sdl.GameControllerOpen(int(event.Which))
		if controller == nil {
			logger.LogError("Failed to open GameController for device index %d", event.Which)
			return
		}
		im.gameControllers = append(im.gameControllers, controller)
		logger.LogInfo("GameController connected (slot %d): %s",
			len(im.gameControllers)-1, controller.Name())
	case sdl.CONTROLLERDEVICEREMOVED:
		i := im.gameControllerSlot(sdl.JoystickID(event.Which))
		if i < 0 {
			return
		}
		c := im.gameControllers[i]
		logger.LogInfo("GameController disconnected (slot %d): %s", i, c.Name())
		c.Close()
		im.gameControllers = append(im.gameControllers[:i], im.gameControllers[i+1:]...)
	}
}

// handleJoyDeviceAdded opens a raw joystick. Skipped for devices SDL
// recognises as a GameController — those are handled by handleControllerDevice
// (and both events fire for the same device).
func (im *InputManager) handleJoyDeviceAdded(event *sdl.JoyDeviceAddedEvent) {
	deviceIndex := int(event.Which)
	if sdl.IsGameController(deviceIndex) {
		return
	}
	if len(im.joysticks) >= maxControllers {
		logger.LogInfo("Ignoring extra joystick (NES supports at most %d)", maxControllers)
		return
	}
	joy := sdl.JoystickOpen(deviceIndex)
	if joy == nil {
		logger.LogError("Failed to open joystick for device index %d", deviceIndex)
		return
	}
	im.joysticks = append(im.joysticks, joy)
	logger.LogInfo("Joystick connected (slot %d): %s (axes=%d buttons=%d hats=%d)",
		len(im.joysticks)-1, joy.Name(), joy.NumAxes(), joy.NumButtons(), joy.NumHats())
}

// handleJoyDeviceRemoved closes a raw joystick. Which is the instance ID.
func (im *InputManager) handleJoyDeviceRemoved(event *sdl.JoyDeviceRemovedEvent) {
	i := im.joystickSlot(sdl.JoystickID(event.Which))
	if i < 0 {
		return
	}
	j := im.joysticks[i]
	logger.LogInfo("Joystick disconnected (slot %d): %s", i, j.Name())
	j.Close()
	im.joysticks = append(im.joysticks[:i], im.joysticks[i+1:]...)
}

// handleKeyboard maps keyboard input to NES controller
func (im *InputManager) handleKeyboard(event *sdl.KeyboardEvent) {
	pressed := event.State == sdl.PRESSED
	input := im.nes.GetInput()

	switch event.Keysym.Sym {
	case sdl.K_z: // A button
		input.SetButton(0, 0, pressed)
	case sdl.K_x: // B button
		input.SetButton(0, 1, pressed)
	case sdl.K_a: // Select
		input.SetButton(0, 2, pressed)
	case sdl.K_s: // Start
		input.SetButton(0, 3, pressed)
	case sdl.K_UP:
		input.SetButton(0, 4, pressed)
	case sdl.K_DOWN:
		input.SetButton(0, 5, pressed)
	case sdl.K_LEFT:
		input.SetButton(0, 6, pressed)
	case sdl.K_RIGHT:
		input.SetButton(0, 7, pressed)
	}
}

// handleJoyButton handles joystick button events
func (im *InputManager) handleJoyButton(event *sdl.JoyButtonEvent) {
	pressed := event.State == sdl.PRESSED
	controllerIndex := im.joystickSlot(event.Which)
	if controllerIndex < 0 {
		return
	}

	input := im.nes.GetInput()

	// Standard gamepad button mapping (works with most controllers)
	switch event.Button {
	case 0: // A button
		input.SetButton(controllerIndex, 0, pressed)
	case 1: // B button
		input.SetButton(controllerIndex, 1, pressed)
	case 2: // X button - also map to B
		input.SetButton(controllerIndex, 1, pressed)
	case 3: // Y button - also map to A
		input.SetButton(controllerIndex, 0, pressed)
	case 8: // Select/Back button
		input.SetButton(controllerIndex, 2, pressed)
	case 9: // Start button
		input.SetButton(controllerIndex, 3, pressed)
	}
}

// handleJoyAxis handles joystick analog stick movements
func (im *InputManager) handleJoyAxis(event *sdl.JoyAxisEvent) {
	controllerIndex := im.joystickSlot(event.Which)
	if controllerIndex < 0 {
		return
	}

	input := im.nes.GetInput()
	deadzone := int16(8000)

	switch event.Axis {
	case 0: // Left stick horizontal
		if event.Value < -deadzone {
			input.SetButton(controllerIndex, 6, true)
			input.SetButton(controllerIndex, 7, false)
		} else if event.Value > deadzone {
			input.SetButton(controllerIndex, 7, true)
			input.SetButton(controllerIndex, 6, false)
		} else {
			input.SetButton(controllerIndex, 6, false)
			input.SetButton(controllerIndex, 7, false)
		}
	case 1: // Left stick vertical
		if event.Value < -deadzone {
			input.SetButton(controllerIndex, 4, true)
			input.SetButton(controllerIndex, 5, false)
		} else if event.Value > deadzone {
			input.SetButton(controllerIndex, 5, true)
			input.SetButton(controllerIndex, 4, false)
		} else {
			input.SetButton(controllerIndex, 4, false)
			input.SetButton(controllerIndex, 5, false)
		}
	}
}

// handleJoyHat handles joystick D-pad events
func (im *InputManager) handleJoyHat(event *sdl.JoyHatEvent) {
	controllerIndex := im.joystickSlot(event.Which)
	if controllerIndex < 0 {
		return
	}

	input := im.nes.GetInput()

	input.SetButton(controllerIndex, 4, event.Value&sdl.HAT_UP != 0)
	input.SetButton(controllerIndex, 5, event.Value&sdl.HAT_DOWN != 0)
	input.SetButton(controllerIndex, 6, event.Value&sdl.HAT_LEFT != 0)
	input.SetButton(controllerIndex, 7, event.Value&sdl.HAT_RIGHT != 0)
}

// handleControllerButton handles SDL GameController button events
func (im *InputManager) handleControllerButton(event *sdl.ControllerButtonEvent) {
	pressed := event.State == sdl.PRESSED
	controllerIndex := im.gameControllerSlot(event.Which)
	if controllerIndex < 0 {
		return
	}

	input := im.nes.GetInput()

	// SDL GameController API provides standardized button mapping
	switch event.Button {
	case sdl.CONTROLLER_BUTTON_A:
		input.SetButton(controllerIndex, 0, pressed)
	case sdl.CONTROLLER_BUTTON_B:
		input.SetButton(controllerIndex, 1, pressed)
	case sdl.CONTROLLER_BUTTON_X:
		input.SetButton(controllerIndex, 1, pressed)
	case sdl.CONTROLLER_BUTTON_Y:
		input.SetButton(controllerIndex, 0, pressed)
	case sdl.CONTROLLER_BUTTON_BACK:
		input.SetButton(controllerIndex, 2, pressed)
	case sdl.CONTROLLER_BUTTON_START:
		input.SetButton(controllerIndex, 3, pressed)
	case sdl.CONTROLLER_BUTTON_DPAD_UP:
		input.SetButton(controllerIndex, 4, pressed)
	case sdl.CONTROLLER_BUTTON_DPAD_DOWN:
		input.SetButton(controllerIndex, 5, pressed)
	case sdl.CONTROLLER_BUTTON_DPAD_LEFT:
		input.SetButton(controllerIndex, 6, pressed)
	case sdl.CONTROLLER_BUTTON_DPAD_RIGHT:
		input.SetButton(controllerIndex, 7, pressed)
	}
}

// handleControllerAxis handles SDL GameController analog stick events
func (im *InputManager) handleControllerAxis(event *sdl.ControllerAxisEvent) {
	controllerIndex := im.gameControllerSlot(event.Which)
	if controllerIndex < 0 {
		return
	}

	input := im.nes.GetInput()
	deadzone := int16(8000)

	switch event.Axis {
	case sdl.CONTROLLER_AXIS_LEFTX:
		if event.Value < -deadzone {
			input.SetButton(controllerIndex, 6, true)
			input.SetButton(controllerIndex, 7, false)
		} else if event.Value > deadzone {
			input.SetButton(controllerIndex, 7, true)
			input.SetButton(controllerIndex, 6, false)
		} else {
			input.SetButton(controllerIndex, 6, false)
			input.SetButton(controllerIndex, 7, false)
		}
	case sdl.CONTROLLER_AXIS_LEFTY:
		if event.Value < -deadzone {
			input.SetButton(controllerIndex, 4, true)
			input.SetButton(controllerIndex, 5, false)
		} else if event.Value > deadzone {
			input.SetButton(controllerIndex, 5, true)
			input.SetButton(controllerIndex, 4, false)
		} else {
			input.SetButton(controllerIndex, 4, false)
			input.SetButton(controllerIndex, 5, false)
		}
	}
}
