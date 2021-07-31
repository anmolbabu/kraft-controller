package utils

import (
	"fmt"
	"strings"
)

type MultiError struct {
	err error
}

func (mErr *MultiError) IsError() bool {
	return mErr.err != nil
}

func (mErr *MultiError) AppendError(err error) {
	if !mErr.IsError() {
		mErr.err = err
		return
	}

	if err != nil {
		mErr.err = fmt.Errorf("%w|%s", mErr.err, err.Error())
	}
}

func (mErr *MultiError) Error() string {
	var errMsg string

	for _, currErrMsg := range strings.Split(mErr.Error(), "|") {
		if errMsg == "" {
			errMsg = currErrMsg
			continue
		}
		errMsg = fmt.Sprintf("%s\n%s", errMsg, currErrMsg)
	}

	return errMsg
}