package circuitbreaker

type CountWindow struct {
	total       uint32 //total最高等于len(bucket)
	slow        uint32
	fail        uint32
	bucketIndex int
	bucket      []OperationResult
}

func (c *CountWindow) Total() uint32 {
	return c.total
}

func (c *CountWindow) Reset() {
	*c = CountWindow{
		bucket: make([]OperationResult, len(c.bucket)),
	}
}

func (c *CountWindow) Push(result OperationResult) {
	// 清理已经存在的bucket, 因为bucket的默认值是 OperationResultUnKnown
	// 所以当bucket是空的时候，不会清理
	switch c.bucket[c.bucketIndex] {
	case OperationResultSuccess:
		c.total--
	case OperationResultSlow:
		c.slow--
		c.total--
	case OperationResultFail:
		c.fail--
		c.total--
	}
	c.total++
	switch result {
	case OperationResultSlow:
		c.slow++
	case OperationResultFail:
		c.fail++
	}
	c.bucket[c.bucketIndex] = result
	c.bucketIndex++
	if c.bucketIndex >= len(c.bucket) {
		c.bucketIndex = 0
	}
}

func (c *CountWindow) FailRate() uint8 {
	return uint8(c.fail * 100 / c.total)
}

func (c *CountWindow) SlowRate() uint8 {
	return uint8(c.slow * 100 / c.total)
}

func NewCountWindow(size uint32) *CountWindow  {
	cw := &CountWindow{
		bucket: make([]OperationResult,size),
	}
	return cw
}