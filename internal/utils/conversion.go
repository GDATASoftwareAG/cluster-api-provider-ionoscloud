package utils

import "strconv"

func ToInt32(s string) (int32, error) {
	if i, err := strconv.ParseInt(s, 10, 32); err != nil {
		return 0, err
	} else {
		return int32(i), nil
	}
}

func ToFloat32(s string) (float32, error) {
	if i, err := strconv.ParseFloat(s, 32); err != nil {
		return 0, err
	} else {
		return float32(i), nil
	}
}
