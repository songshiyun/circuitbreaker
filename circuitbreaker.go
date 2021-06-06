package circuitbreaker

import (
	"sync"
	"time"
)

const (
	OperationResultUnKnown OperationResult = iota
	OperationResultSuccess
	OperationResultSlow
	OperationResultFail
)

const (
	StateClosed = iota + 1
	StateHalfOpen
	StateOpen
)

var StateDescMap = map[int]string{
	StateClosed:   "Closed",
	StateHalfOpen: "HalfOpen",
	StateOpen:     "Open",
}

type OperationResult uint8
type State uint8

type Window interface {
	Total() uint32
	Reset()
	Push(result OperationResult)
	FailRate() uint8
	SlowRate() uint8
}

type CircuitBreaker struct {
	lock            sync.Mutex
	conf            *Config
	state           State
	transitTime     time.Time
	window          Window
	callsInHalfOpen uint32
	logicCounter    uint32
	listener        EventListenerFunc
}

func New(conf *Config) *CircuitBreaker {
	cb := &CircuitBreaker{conf: conf}
	cb.transferStat(StateClosed)
	return cb
}

func (cb *CircuitBreaker) SetState(state State) {
	cb.lock.Lock()
	defer cb.lock.Unlock()
	cb.transferStat(state)
}

func (cb *CircuitBreaker) GetState() State {
	return cb.state
}

func (cb *CircuitBreaker) SetEventListener(listener EventListenerFunc) {
	cb.lock.Lock()
	defer cb.lock.Unlock()
	cb.listener = listener
}

func (cb *CircuitBreaker) Acquire() (bool, uint32) {
	cb.lock.Lock()
	defer cb.lock.Unlock()
	//closed状态总是允许
	if cb.state == StateClosed {
		return true, cb.logicCounter
	}
	// 当处在Open状态的时间间隔小于DurationInOpen，否则将状态转换至halfOpen
	// https://docs.microsoft.com/en-us/previous-versions/msp-n-p/dn589784(v=pandp.10)?redirectedfrom=MSDN
	now := time.Now()
	if cb.state == StateOpen {
		if now.Sub(cb.transitTime) < cb.conf.DurationInOpen {
			return false, cb.logicCounter
		}
		cb.transferStat(StateHalfOpen)
	}
	// 处于halfOpen状态
	if cb.callsInHalfOpen < cb.conf.HalfOpenCalls {
		cb.callsInHalfOpen++
		return true, cb.logicCounter
	}
	// 如果超过了MaxDurationInHalfOpen时间间隔仍然处于halfOpen状态，则回退到Open状态
	// diff 微软的是处于halfOpen状态，在这个状态中的所有请求都成功了，则状态切换到Closed
	// 如果在halfOpen状态有任何失败则回退到Open状态
	if cb.conf.MaxDurationInHalfOpen > 0 && now.Sub(cb.transitTime) > cb.conf.MaxDurationInHalfOpen {
		cb.transferStat(StateHalfOpen)
	}
	return false, cb.logicCounter
}

func (cb *CircuitBreaker) SetResult(counter uint32, hasErr bool, duration time.Duration) {
	res := OperationResultSuccess
	if hasErr {
		res = OperationResultFail
	} else if duration > cb.conf.SlowCallDuration {
		res = OperationResultSlow
	}
	cb.lock.Lock()
	defer cb.lock.Unlock()
	//steal
	if cb.logicCounter != counter {
		return
	}
	cb.window.Push(res)
	minCalls := cb.conf.MinNumOfCalls
	if cb.state == StateHalfOpen {
		if minCalls > cb.conf.HalfOpenCalls {
			minCalls = cb.conf.HalfOpenCalls
		}
	}
	if cb.window.Total() < minCalls {
		return
	}
	if rate := cb.window.FailRate(); rate >= cb.conf.FailRate {
		cb.transferStat(StateOpen)
	} else if rate := cb.window.SlowRate(); rate >= cb.conf.SlowCallRate {
		cb.transferStat(StateOpen)
	} else if cb.state == StateHalfOpen {
		cb.transferStat(StateClosed)
	}
}

func (cb *CircuitBreaker) transferStat(state State) {
	old := cb.state
	if old == state {
		return
	}
	cb.state = state
	cb.transitTime = time.Now()
	cb.logicCounter++
	if state == StateClosed {
		cb.window = NewCountWindow(cb.conf.WindowSize)
	} else if state == StateHalfOpen {
		cb.window = NewCountWindow(cb.conf.WindowSize)
		cb.callsInHalfOpen = 0
	}
	if cb.listener != nil {
		event := &Event{
			Time:    cb.transitTime,
			OldStat: uint8(old),
			NewStat: uint8(state),
		}
		go cb.listener(event)
	}
}
