package hub

import (
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10

	// maxMessageSize : ขนาดข้อความขาเข้าสูงสุด
	// ต้องใหญ่พอรับ WebRTC SDP offer/answer (ปกติ 1–4 KB) ไม่งั้นพอเปิดกล้อง
	// ข้อความ signal จะเกินลิมิต → gorilla ปิด connection (1009) → สายหลุด
	// 64 KB เผื่อ SDP ที่มี codec/candidate เยอะ (เป็น control message ไม่ใช่สื่อ — สื่อวิ่ง P2P)
	maxMessageSize = 64 * 1024
)

// Client แทนการเชื่อมต่อของผู้เล่น 1 คน — "ท่อ" รับส่ง byte เท่านั้น
// ไม่มี game state (ชื่อ/ตำแหน่ง) อีกแล้ว — ย้ายไปอยู่ room.Manager
//   - id   : รหัสประจำตัว (hub แจกตอนต่อ)
//   - room : ห้องที่อยู่ (ใช้ route ข้อความ)
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
	id   string
	room string
}

// readPump อ่านข้อความขาเข้า → ส่งต่อให้ hub.incoming (ไม่ตีความเอง)
// router จะเป็นคนแกะ + ตัดสินใจว่าทำอะไร
func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		// ส่งต่อให้ router ผ่าน channel กลางของ hub (พร้อมแปะว่าใคร/ห้องไหน)
		c.hub.incoming <- Inbound{ClientID: c.id, Room: c.room, Data: message}
	}
}

// writePump ดึงจาก send → เขียนลง socket + ส่ง ping เป็นระยะ
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
