package main

import (
	"context"
	"encoding/json"

	"github.com/libp2p/go-libp2p/core/peer"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

// Chatroom 的 Messages channel 的緩衝大小
const ChatRoomBufSize = 128

type ChatRoom struct {
	Messages chan *ChatMessage

	ctx   context.Context
	ps    *pubsub.PubSub
	topic *pubsub.Topic
	sub   *pubsub.Subscription

	roomName string
	self     peer.ID
	nick     string
}

type ChatMessage struct {
	Message    string
	SenderID   string
	SenderNick string
}

// 寄送 message 給 pubsub topic
func (cr *ChatRoom) Publish(message string) error {
	m := ChatMessage{
		Message:    message,
		SenderID:   cr.self.Pretty(),
		SenderNick: cr.nick,
	}

	msgBytes, err := json.Marshal(m)
	if err != nil {
		return err
	}

	return cr.topic.Publish(cr.ctx, msgBytes)
}

func (cr *ChatRoom) readLoop() {
	for {
		msg, err := cr.sub.Next(cr.ctx)
		if err != nil {
			close(cr.Messages)
		}

		// 忽略自己發布的訊息
		if msg.ReceivedFrom == cr.self {
			continue
		}

		cm := new(ChatMessage)
		err = json.Unmarshal(msg.Data, cm) // Unmarshal 的訊息會被存在 cm
		if err != nil {
			continue
		}

		// 把訊息送到 chatroom 的 Messages channel
		cr.Messages <- cm
	}
}

// 這個 function 會去訂閱一個 Pubsub topic， topic 就是聊天室的名字
// 成功的話會回傳一個 ChatRoom 物件
func JoinChatRoom(ctx context.Context, ps *pubsub.PubSub, selfID peer.ID, nickname string, roomName string) (*ChatRoom, error) {
	// 加入 topic
	topic, err := ps.Join("chat-room:" + roomName)
	if err != nil {
		return nil, err
	}

	// 訂閱 topic
	sub, err := topic.Subscribe()
	if err != nil {
		return nil, err
	}

	cr := &ChatRoom{
		ctx:      ctx,
		ps:       ps,
		topic:    topic,
		sub:      sub,
		self:     selfID,
		nick:     nickname,
		roomName: roomName,
		Messages: make(chan *ChatMessage, ChatRoomBufSize),
	}

	go cr.readLoop() // 一直持續把新訊息加入 channel

	return cr, nil
}
