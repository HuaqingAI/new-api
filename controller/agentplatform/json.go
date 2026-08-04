package agentplatform

func jsonTextToRawMessage(value string) []byte {
	if value == "" {
		return nil
	}
	return []byte(value)
}
