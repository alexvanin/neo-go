// Code generated by "stringer -type=CommandType -output=message_string.go"; DO NOT EDIT.

package network

import "strconv"

func _() {
	// An "invalid array index" compiler error signifies that the constant values have changed.
	// Re-run the stringer command to generate them again.
	var x [1]struct{}
	_ = x[CMDVersion-0]
	_ = x[CMDVerack-1]
	_ = x[CMDGetAddr-16]
	_ = x[CMDAddr-17]
	_ = x[CMDPing-24]
	_ = x[CMDPong-25]
	_ = x[CMDGetHeaders-32]
	_ = x[CMDHeaders-33]
	_ = x[CMDGetBlocks-36]
	_ = x[CMDMempool-37]
	_ = x[CMDInv-39]
	_ = x[CMDGetData-40]
	_ = x[CMDGetBlockByIndex-41]
	_ = x[CMDNotFound-42]
	_ = x[CMDTX-43]
	_ = x[CMDBlock-44]
	_ = x[CMDExtensible-46]
	_ = x[CMDP2PNotaryRequest-80]
	_ = x[CMDReject-47]
	_ = x[CMDFilterLoad-48]
	_ = x[CMDFilterAdd-49]
	_ = x[CMDFilterClear-50]
	_ = x[CMDMerkleBlock-56]
	_ = x[CMDAlert-64]
}

const (
	_CommandType_name_0 = "CMDVersionCMDVerack"
	_CommandType_name_1 = "CMDGetAddrCMDAddr"
	_CommandType_name_2 = "CMDPingCMDPong"
	_CommandType_name_3 = "CMDGetHeadersCMDHeaders"
	_CommandType_name_4 = "CMDGetBlocksCMDMempool"
	_CommandType_name_5 = "CMDInvCMDGetDataCMDGetBlockByIndexCMDNotFoundCMDTXCMDBlock"
	_CommandType_name_6 = "CMDExtensibleCMDRejectCMDFilterLoadCMDFilterAddCMDFilterClear"
	_CommandType_name_7 = "CMDMerkleBlock"
	_CommandType_name_8 = "CMDAlert"
	_CommandType_name_9 = "CMDP2PNotaryRequest"
)

var (
	_CommandType_index_0 = [...]uint8{0, 10, 19}
	_CommandType_index_1 = [...]uint8{0, 10, 17}
	_CommandType_index_2 = [...]uint8{0, 7, 14}
	_CommandType_index_3 = [...]uint8{0, 13, 23}
	_CommandType_index_4 = [...]uint8{0, 12, 22}
	_CommandType_index_5 = [...]uint8{0, 6, 16, 34, 45, 50, 58}
	_CommandType_index_6 = [...]uint8{0, 13, 22, 35, 47, 61}
)

func (i CommandType) String() string {
	switch {
	case i <= 1:
		return _CommandType_name_0[_CommandType_index_0[i]:_CommandType_index_0[i+1]]
	case 16 <= i && i <= 17:
		i -= 16
		return _CommandType_name_1[_CommandType_index_1[i]:_CommandType_index_1[i+1]]
	case 24 <= i && i <= 25:
		i -= 24
		return _CommandType_name_2[_CommandType_index_2[i]:_CommandType_index_2[i+1]]
	case 32 <= i && i <= 33:
		i -= 32
		return _CommandType_name_3[_CommandType_index_3[i]:_CommandType_index_3[i+1]]
	case 36 <= i && i <= 37:
		i -= 36
		return _CommandType_name_4[_CommandType_index_4[i]:_CommandType_index_4[i+1]]
	case 39 <= i && i <= 44:
		i -= 39
		return _CommandType_name_5[_CommandType_index_5[i]:_CommandType_index_5[i+1]]
	case 46 <= i && i <= 50:
		i -= 46
		return _CommandType_name_6[_CommandType_index_6[i]:_CommandType_index_6[i+1]]
	case i == 56:
		return _CommandType_name_7
	case i == 64:
		return _CommandType_name_8
	case i == 80:
		return _CommandType_name_9
	default:
		return "CommandType(" + strconv.FormatInt(int64(i), 10) + ")"
	}
}
