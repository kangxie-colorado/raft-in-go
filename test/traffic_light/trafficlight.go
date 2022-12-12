package trafficlight

import (
	"fmt"
	"strconv"
	"time"

	"github.com/enriquebris/goconcurrentqueue"
	"github.com/kangxie-colorado/golang-primer/messaging/lib"
	log "github.com/sirupsen/logrus"

	"github.com/Workiva/go-datastructures/queue"
	"github.com/Workiva/go-datastructures/set"
)

type TrafficLightColor int

const (
	Undefined TrafficLightColor = iota
	Green
	Yellow
	Red
)

const YellowTimer int = 5

type TrafficLight struct {
	whichIsGreen int
	lightColors  [2]TrafficLightColor
	Timer        int    // how much time has elapsed in this state(color is just state)
	greenSecs    [2]int // how long to stay green

	InputQueue *goconcurrentqueue.FIFO
	Controller LightControl

	MySock         lib.SocketDescriptor
	ButtonListener func(*TrafficLight) // use a callback is good enough
}

func CreateATrafficLight(greenSecs [2]int, lightSocks [2]lib.SocketDescriptor, mySock lib.SocketDescriptor) *TrafficLight {
	var tfl = TrafficLight{}
	// some light has to start green, otherwise, it is deadlock...
	// or use a red timer and let the short red timer to fire off and become green... not a big deal
	// only implementation details
	tfl.whichIsGreen = 0 // let the first light to be green, first light? North-South?
	tfl.lightColors = [2]TrafficLightColor{Green, Red}
	tfl.Timer = 0
	tfl.greenSecs = greenSecs
	tfl.InputQueue = goconcurrentqueue.NewFIFO()
	tfl.Controller = CreateLightControl(lightSocks)

	tfl.ButtonListener = StartButtonListener_old
	tfl.MySock = mySock

	return &tfl
}

func (tfl *TrafficLight) StartTimer() {
	for {
		time.Sleep(1000 * time.Millisecond)
		tfl.InputQueue.Enqueue("One Second Passed")
	}
}

func (tfl *TrafficLight) Start() {

	go tfl.ButtonListener(tfl)
	go tfl.StartTimer()

	// main loop of traffic ligh state machine
	for {
		theOtherLightNo := 0
		if tfl.whichIsGreen == 0 {
			theOtherLightNo = 1
		}

		msg, err := tfl.InputQueue.DequeueOrWaitForNextElement()
		if err != nil {
			log.Errorln("Error dequeue inbox", err.Error())
		}

		switch msg {
		case "Button Pressed":
			if tfl.lightColors[tfl.whichIsGreen] == Green && tfl.greenSecs[tfl.whichIsGreen] == 60 {
				if tfl.Timer > 30 {
					tfl.Controller.changeColor(tfl.whichIsGreen, Yellow, "5")
					tfl.lightColors[tfl.whichIsGreen] = Yellow
					tfl.Timer = 0
				} else {
					tfl.Timer += 30
				}

			}
		case "One Second Passed":
			// one more second spent in this state
			tfl.Timer += 1

			if tfl.lightColors[tfl.whichIsGreen] == Green && tfl.Timer > tfl.greenSecs[tfl.whichIsGreen] {
				log.Infof("Lights colors are %v, turning light%v from Green to Yellow", tfl.lightColors, tfl.whichIsGreen)
				tfl.Controller.changeColor(tfl.whichIsGreen, Yellow, "5")
				tfl.lightColors[tfl.whichIsGreen] = Yellow
				tfl.Timer = 0
			} else if tfl.lightColors[tfl.whichIsGreen] == Yellow && tfl.Timer > YellowTimer {

				log.Infof("Lights colors are %v, turning light%v from Yellow to Red", tfl.lightColors, tfl.whichIsGreen)
				tfl.Controller.changeColor(tfl.whichIsGreen, Red, "")
				tfl.lightColors[tfl.whichIsGreen] = Red

				// this should change another light to green, how to do that...
				// huh, actually there is not another state machine, one state machine controls two lights...
				// yeah, and shit, I entered a dead place when thinking I need two timers for green/red light length

				tfl.Controller.changeColor(theOtherLightNo, Green, strconv.Itoa(tfl.greenSecs[theOtherLightNo]))
				tfl.lightColors[theOtherLightNo] = Green
				tfl.whichIsGreen = theOtherLightNo

				tfl.Timer = 0
			} else {
				if tfl.lightColors[tfl.whichIsGreen] == Green {
					tfl.Controller.changeColor(tfl.whichIsGreen, Green, strconv.Itoa(tfl.greenSecs[tfl.whichIsGreen]-tfl.Timer))

				} else if tfl.lightColors[tfl.whichIsGreen] == Yellow {
					tfl.Controller.changeColor(tfl.whichIsGreen, Yellow, strconv.Itoa(YellowTimer-tfl.Timer))
				}
			}

		default:
			log.Errorln("Unknown input events:", msg)
		}
	}

}

// refactoring: try to isolate the state machine out of the controller

type TFLStateMachineInput struct {
	timerElapsed  bool
	buttonPressed bool
}

type TFLStateMachine struct {
	whichIsGreen  int
	lightColors   [2]TrafficLightColor
	timer         int    // how much time has elapsed in this state(color is just state)
	greenSecs     [2]int // how long to stay green
	buttonPressed bool
}

type TFLStateMachineOutput struct {
	colors     [2]TrafficLightColor
	countDowns [2]string
}

// caller allocates the memory
func (tflsm *TFLStateMachine) initTFLSM(greenSecs [2]int) {
	tflsm.whichIsGreen = 0
	tflsm.lightColors = [2]TrafficLightColor{Green, Red}
	tflsm.timer = 0
	tflsm.greenSecs = greenSecs
}

func (tflsm *TFLStateMachine) asString() string {
	return fmt.Sprintf("TFLStateMachine{whichIsGreen=%v, lightColors=%v, timer=%v, greenSecs=%v, buttonPressed=%v}",
		tflsm.whichIsGreen,
		tflsm.lightColors,
		tflsm.timer,
		tflsm.greenSecs,
		tflsm.buttonPressed,
	)
}

func (tflsm *TFLStateMachine) stateMachineRun(input TFLStateMachineInput) *TFLStateMachineOutput {
	theOtherLightNo := 0
	if tflsm.whichIsGreen == 0 {
		theOtherLightNo = 1
	}

	// little bug: this initialize colors to 0?
	// every time press the button it falshes red?
	output := &TFLStateMachineOutput{}
	// notice output when intialized, colors field is initialzied to all zero
	// at least, copy the state machine colors in
	output.colors = tflsm.lightColors

	// only need to manipulate the green/yellow stuff, where yellow is transition state of green (identified by whichIsGreen until it turns red)
	// but the point is not to worry about red, that is just derived from the green/yellow changes...
	if input.buttonPressed {
		if tflsm.lightColors[tflsm.whichIsGreen] == Green && tflsm.greenSecs[tflsm.whichIsGreen] == 60 && !tflsm.buttonPressed {
			if tflsm.timer > 30 {
				output.colors[tflsm.whichIsGreen] = Yellow
				output.countDowns[tflsm.whichIsGreen] = strconv.Itoa(YellowTimer)

				tflsm.lightColors[tflsm.whichIsGreen] = Yellow
				tflsm.timer = 0
			} else {
				tflsm.timer += 30
				currentCountDown := tflsm.greenSecs[tflsm.whichIsGreen] - tflsm.timer
				output.countDowns[tflsm.whichIsGreen] = strconv.Itoa(currentCountDown)

			}

			tflsm.buttonPressed = true
		}
	}

	if input.timerElapsed {
		tflsm.timer += 1

		if tflsm.lightColors[tflsm.whichIsGreen] == Green && tflsm.timer > tflsm.greenSecs[tflsm.whichIsGreen] {
			log.Infof("Lights colors are %v, turning light%v from Green to Yellow", tflsm.lightColors, tflsm.whichIsGreen)
			output.colors[tflsm.whichIsGreen] = Yellow
			output.countDowns[tflsm.whichIsGreen] = strconv.Itoa(YellowTimer)

			tflsm.lightColors[tflsm.whichIsGreen] = Yellow
			tflsm.timer = 0
		} else if tflsm.lightColors[tflsm.whichIsGreen] == Yellow && tflsm.timer > YellowTimer {

			log.Infof("Lights colors are %v, turning light%v from Yellow to Red", tflsm.lightColors, tflsm.whichIsGreen)

			output.colors[tflsm.whichIsGreen] = Red
			tflsm.lightColors[tflsm.whichIsGreen] = Red

			// this should change another light to green, how to do that...
			// huh, actually there is not another state machine, one state machine controls two lights...
			// yeah, and shit, I entered a dead place when thinking I need two timers for green/red light length
			output.colors[theOtherLightNo] = Green
			output.countDowns[theOtherLightNo] = strconv.Itoa(tflsm.greenSecs[theOtherLightNo])
			tflsm.lightColors[theOtherLightNo] = Green

			tflsm.whichIsGreen = theOtherLightNo

			tflsm.timer = 0
			tflsm.buttonPressed = false
		} else {
			if tflsm.lightColors[tflsm.whichIsGreen] == Green {
				output.colors[tflsm.whichIsGreen] = Green
				output.countDowns[tflsm.whichIsGreen] = strconv.Itoa(tflsm.greenSecs[tflsm.whichIsGreen] - tflsm.timer)
			} else if tflsm.lightColors[tflsm.whichIsGreen] == Yellow {
				output.colors[tflsm.whichIsGreen] = Yellow
				output.countDowns[tflsm.whichIsGreen] = strconv.Itoa(YellowTimer - tflsm.timer)

			}
		}
	}

	return output
}

type TFLControl struct {
	stateMachine *TFLStateMachine

	inputQueue *goconcurrentqueue.FIFO
	controller LightControl

	mySock         lib.SocketDescriptor
	buttonListener func(*TFLControl) // use a callback is good enough
}

func CreateTFLControl(greenSecs [2]int, lightSocks [2]lib.SocketDescriptor, mySock lib.SocketDescriptor) *TFLControl {
	var tflctl = TFLControl{}

	sm := TFLStateMachine{}
	sm.initTFLSM(greenSecs)
	tflctl.stateMachine = &sm

	tflctl.inputQueue = goconcurrentqueue.NewFIFO()
	tflctl.controller = CreateLightControl(lightSocks)

	tflctl.buttonListener = StartButtonListener
	tflctl.mySock = mySock

	return &tflctl
}

func (tflctl *TFLControl) startTimer() {
	for {
		time.Sleep(1000 * time.Millisecond)
		tflctl.inputQueue.Enqueue("One Second Passed")
	}
}

func (tflctl *TFLControl) controlTheLights(output *TFLStateMachineOutput) {
	for i := 0; i < len(output.colors); i++ {
		tflctl.controller.changeColor(i, output.colors[i], output.countDowns[i])
	}
}

func (tflctl *TFLControl) Start() {
	go tflctl.buttonListener(tflctl)
	go tflctl.startTimer()

	for {
		msg, err := tflctl.inputQueue.DequeueOrWaitForNextElement()
		if err != nil {
			log.Errorln("Error dequeue inbox", err.Error())
		}

		buttonPressed := false
		timerElapsed := false
		switch msg {
		case "Button Pressed":
			buttonPressed = true
		case "One Second Passed":
			timerElapsed = true
		default:
			log.Errorln("Unknown input events:", msg)
		}

		output := tflctl.stateMachine.stateMachineRun(TFLStateMachineInput{timerElapsed: timerElapsed, buttonPressed: buttonPressed})
		if output != nil {
			tflctl.controlTheLights(output)
		}

	}

}

func TestTFLStateMachine() {
	sm := TFLStateMachine{}
	sm.initTFLSM([2]int{30, 60})

	toTestQueue := queue.New(100)
	toTestQueue.Put(sm)
	seen := set.New()

	buttonPressed := [2]bool{false, true}
	timerElapsed := [2]bool{false, true}

	for !toTestQueue.Empty() {
		item, err := toTestQueue.Get(1)
		if err != nil {
			log.Errorln("Error getting from toTestQueue", err.Error())
		}

		/**
		// can convert interface to stuct back this way
		var state TFLStateMachine
		err = mapstructure.Decode(item[0], &state)
		if err != nil {
			log.Errorln("Error decode state", err.Error())

		}
		**/
		// or this way
		state := item[0].(TFLStateMachine)

		if !seen.Exists(state.asString()) {
			seen.Add(state.asString())

			for _, bp := range buttonPressed {
				for _, te := range timerElapsed {
					stateCpy := state
					stateCpy.stateMachineRun(TFLStateMachineInput{buttonPressed: bp, timerElapsed: te})
					toTestQueue.Put(stateCpy)
				}
			}

		}

		log.Infof("%v states are tested by far", seen.Len())

	}
	for _, item := range seen.Flatten() {
		log.Infof("%v\n", item)
	}

}
