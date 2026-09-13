package main

import (
	"ai-server/internal/protocol"
	"ai-server/internal/session"
	"time"
)

// Tolerate clock quantization and scheduling variance without granting earlier gameplay.
// Late completion is unrestricted. This margin is fixed, not banked across rounds.
const playbackEarlyGrace = 150 * time.Millisecond

func handleRealtimePacketOnWire(store *session.Store, packet protocol.Packet, socket *packetSession) (packetResult, string) {
	result, violation := handleRealtimePacket(store, packet, socket, time.Now())
	for violation == "" && !result.realtimeResumeAt.IsZero() {
		// One event at the authoritative deadline, not a retry interval or client-duration timeout.
		// Keep this connection's input ordered while releasing the shared battle lock for teammates.
		deadline := time.NewTimer(time.Until(result.realtimeResumeAt))
		<-deadline.C
		// Revalidate: another teammate may have extended/consumed the batch while this ACK waited.
		result, violation = handleRealtimeGameplay(store, packet, socket, time.Now())
	}
	return result, violation
}
