package circuitbreaker

import (
	"testing"
	"time"
)

func TestNewCountWindow(t *testing.T) {
	conf := Config{
		FailRate:              50,
		SlowCallRate:          60,
		WindowSize:            20,
		HalfOpenCalls:         5,
		MinNumOfCalls:         10,
		SlowCallDuration:      time.Millisecond * 10,
		MaxDurationInHalfOpen: 5 * time.Second,
		DurationInOpen:        5 * time.Second,
	}
	cb := New(&conf)
	runSharedCases(t,cb)
	cb.SetState(StateClosed)
	// 插入12条成功记录
	for i := 0; i < 12; i++ {
		if permitted, stateID := cb.Acquire(); !permitted {
			t.Errorf("acquire permission should succeeded, i = %d", i)
		} else if cb.GetState() != StateClosed {
			t.Errorf("circuit breaker state should be Closed")
		} else {
			cb.SetResult(stateID, false, time.Millisecond)
		}
	}
	// 插入10条失败记录
	for i := 0; i < 10; i++ {
		if permitted, stateID := cb.Acquire(); !permitted {
			t.Errorf("acquire permission should succeeded, i = %d", i)
		} else if cb.GetState() != StateClosed {
			t.Errorf("circuit breaker state should be Closed")
		} else {
			cb.SetResult(stateID, true, time.Millisecond)
		}
	}
	// 现在的状态应该是open
	if cb.GetState() != StateOpen {
		t.Errorf("circuit breaker state should be Open")
	}
}

func runSharedCases(t *testing.T, cb *CircuitBreaker) {
	// 插入10次成功，这个时候状态应该是closed
	for i := 0; i < 10; i++ {
		if permitted, stateID := cb.Acquire(); permitted {
			cb.SetResult(stateID, false, time.Millisecond)
		} else {
			t.Errorf("acquire permission should succeeded, i = %d", i)
		}
	}
	// 插入10 次失败，状态应该是opened
	for i := 0; i < 10; i++ {
		if permitted, stateID := cb.Acquire(); permitted {
			cb.SetResult(stateID, true, time.Millisecond)
		} else {
			t.Errorf("acquire permission should succeeded, i = %d", i)
		}
	}

	if cb.GetState() != StateOpen {
		t.Errorf("circuit breaker state should be Open")
	}
	// open状态并且没有超时，不应该获取成功
	if permitted, _ := cb.Acquire(); permitted {
		t.Errorf("acquire permission should fail")
	}

	// 等等五秒，open->halfOpen
	time.Sleep(5* time.Second)

	// 插入五次成功，状态应该最终变成closed，状态应该有halfOpen->Closed
	for i := 0; i < 5; i++ {
		if permitted, stateID := cb.Acquire(); !permitted {
			t.Errorf("acquire permission should succeeded, i = %d", i)
		} else if cb.GetState() != StateHalfOpen {
			t.Errorf("circuit breaker state should be HalfOpen")
		} else {
			cb.SetResult(stateID, false, time.Millisecond)
		}
	}
	if cb.GetState() != StateClosed {
		t.Errorf("circuit breaker state should be Closed")
	}
	// 这个时候应该获取成功
	if permitted, _ := cb.Acquire(); !permitted {
		t.Errorf("acquire permission should succeeded")
	}
	// 插入8次成功
	for i := 0; i < 8; i++ {
		if permitted, stateID := cb.Acquire(); permitted {
			cb.SetResult(stateID, false, time.Millisecond)
		} else {
			t.Errorf("acquire permission should succeeded, i = %d", i)
		}
	}
	// 插入12次超时，这个时候状态应该是Opened
	for i := 0; i < 12; i++ {
		if permitted, stateID := cb.Acquire(); permitted {
			cb.SetResult(stateID, false, 11*time.Millisecond)
		} else {
			t.Errorf("acquire permission should succeeded, i = %d", i)
		}
	}

	if cb.GetState() != StateOpen {
		t.Errorf("circuit breaker state should be Open")
	}
	if permitted, _ := cb.Acquire(); permitted {
		t.Errorf("acquire permission should fail")
	}
	// 等待5秒，状态变成 halfOpen
	time.Sleep(5 * time.Second)

	//插入五次slow result, halfOpen->Opened
	for i := 0; i < 5; i++ {
		if permitted, stateID := cb.Acquire(); !permitted {
			t.Errorf("acquire permission should succeeded, i = %d", i)
		} else if cb.GetState() != StateHalfOpen {
			t.Errorf("circuit breaker state should be HalfOpen")
		} else {
			cb.SetResult(stateID, false, 11*time.Millisecond)
		}
	}
	if cb.GetState() != StateOpen {
		t.Errorf("circuit breaker state should be Open")
	}
	if permitted, _ := cb.Acquire(); permitted {
		t.Errorf("acquire permission should fail")
	}
}
