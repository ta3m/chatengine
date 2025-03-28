// Copyright 2022 Teamgram Authors
//  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Author: teamgramio (teamgram.io@gmail.com)
//

package core

import (
	"time"

	"github.com/teamgram/proto/mtproto"
	msgpb "github.com/teamgram/teamgram-server/app/messenger/msg/msg/msg"

	"github.com/zeromicro/go-zero/core/contextx"
	"github.com/zeromicro/go-zero/core/threading"
)

// MessagesSendMultiMedia
// messages.sendMultiMedia#f803138f flags:# silent:flags.5?true background:flags.6?true clear_draft:flags.7?true noforwards:flags.14?true peer:InputPeer reply_to_msg_id:flags.0?int multi_media:Vector<InputSingleMedia> schedule_date:flags.10?int send_as:flags.13?InputPeer = Updates;
func (c *MessagesCore) MessagesSendMultiMedia(in *mtproto.TLMessagesSendMultiMedia) (*mtproto.Updates, error) {
	// peer
	var (
		peer       *mtproto.PeerUtil
		linkChatId int64
		err        error
	)

	peer = mtproto.FromInputPeer2(c.MD.UserId, in.Peer)
	switch peer.PeerType {
	case mtproto.PEER_SELF:
		peer.PeerType = mtproto.PEER_USER
	case mtproto.PEER_USER:
		//if !md.IsBot {
		//	hasBot = s.UserFacade.IsBot(ctx, peer.PeerId)
		//}
	case mtproto.PEER_CHAT:
	case mtproto.PEER_CHANNEL:
		//channel, _ := s.ChannelFacade.GetMutableChannel(ctx, peer.PeerId, md.UserId)
		//if channel != nil && channel.Channel.LinkedChatId > 0 {
		//	linkChatId = channel.Channel.LinkedChatId
		//}
	default:
		c.Logger.Errorf("invalid peer: %v", in.Peer)
		err = mtproto.ErrPeerIdInvalid
		return nil, err
	}

	// 1. draft
	//if request.GetClearDraft() {
	//	go func() {
	//		s.doClearDraft(ctx, md.UserId, md.AuthId, peer)
	//	}()
	//}

	/////////////////////////////////////////////////////////////////////////////////////
	/*
		messages.sendMultiMedia

		flags	#	Flags, see TL conditional fields
		silent	flags.5?true	Whether to send the album silently (no notification triggered)
		background	flags.6?true	Send in background?
		clear_draft	flags.7?true	Whether to clear drafts
		peer	InputPeer	The destination chat
		reply_to_msg_id	flags.0?int	The message to reply to
		multi_media	Vector<InputSingleMedia>	The medias to send
		schedule_date	flags.10?int	Scheduled message date for scheduled messages
	*/

	groupedId := c.svcCtx.Dao.IDGenClient2.NextId(c.ctx)
	//, &idgen.TLIdgenNextId{
	//	Constructor:          0,
	//	XXX_NoUnkeyedLiteral: struct{}{},
	//	XXX_unrecognized:     nil,
	//	XXX_sizecache:        0,
	//}) & types.Int64Value{Value: idgen.GetUUID()}
	outboxMultiMedia := make([]*msgpb.OutboxMessage, 0, len(in.MultiMedia))
	for _, media := range in.MultiMedia {
		/*
			inputSingleMedia

			media	InputMedia	The media
			random_id	long	Unique client media ID required to prevent message resending
			message	string	A caption for the media
			entities	flags.0?Vector<MessageEntity>	Message entities for styled text
		*/
		if len(media.Message) > 4000 {
			err := mtproto.ErrMediaCaptionTooLong
			c.Logger.Errorf("messages.sendMultiMedia: %v", err)
			return nil, err
		}

		outMessage := mtproto.MakeTLMessage(&mtproto.Message{
			Out:                  true,
			Mentioned:            false,
			MediaUnread:          false,
			Silent:               in.Silent,
			Post:                 false,
			FromScheduled:        false,
			Legacy:               false,
			EditHide:             false,
			Pinned:               false,
			Noforwards:           in.Noforwards,
			InvertMedia:          in.InvertMedia,
			Id:                   0,
			FromId:               mtproto.MakePeerUser(c.MD.UserId),
			PeerId:               peer.ToPeer(),
			SavedPeerId:          nil,
			FwdFrom:              nil,
			ViaBotId:             nil,
			ReplyTo:              nil,
			Date:                 int32(time.Now().Unix()),
			Message:              media.Message,
			Media:                nil,
			ReplyMarkup:          nil, // request.ReplyMarkup,
			Entities:             media.Entities,
			Views:                nil,
			Forwards:             nil,
			Replies:              nil,
			EditDate:             nil,
			PostAuthor:           nil,
			GroupedId:            mtproto.MakeFlagsInt64(groupedId),
			Reactions:            nil,
			RestrictionReason:    nil,
			TtlPeriod:            nil,
			QuickReplyShortcutId: nil,
			Effect:               in.Effect,
			Factcheck:            nil,
		}).To_Message()

		// Fix SavedPeerId
		if peer.IsSelfUser(c.MD.UserId) {
			outMessage.SavedPeerId = peer.ToPeer()
		}

		// Fix ReplyToMsgId
		if in.GetReplyToMsgId() != nil {
			outMessage.ReplyTo = mtproto.MakeTLMessageReplyHeader(&mtproto.MessageReplyHeader{
				ReplyToMsgId:           in.GetReplyToMsgId().GetValue(),
				ReplyToMsgId_INT32:     in.GetReplyToMsgId().GetValue(),
				ReplyToMsgId_FLAGINT32: in.GetReplyToMsgId(),
				ReplyToPeerId:          nil,
				ReplyToTopId:           nil,
			}).To_MessageReplyHeader()
		} else if in.GetReplyTo() != nil {
			switch in.ReplyTo.PredicateName {
			case mtproto.Predicate_inputReplyToMessage:
				outMessage.ReplyTo = mtproto.MakeTLMessageReplyHeader(&mtproto.MessageReplyHeader{
					ReplyToMsgId:           in.GetReplyTo().GetReplyToMsgId(),
					ReplyToMsgId_INT32:     in.GetReplyTo().GetReplyToMsgId(),
					ReplyToMsgId_FLAGINT32: mtproto.MakeFlagsInt32(in.GetReplyTo().GetReplyToMsgId()),
					ReplyToPeerId:          nil,
					ReplyToTopId:           nil,
				}).To_MessageReplyHeader()
				if in.GetReplyTo().GetQuoteText() != nil {
					outMessage.ReplyTo.Quote = true
					outMessage.ReplyTo.QuoteText = in.GetReplyTo().GetQuoteText()
					outMessage.ReplyTo.QuoteEntities = in.GetReplyTo().GetQuoteEntities()
					outMessage.ReplyTo.QuoteOffset = in.GetReplyTo().GetQuoteOffset()
				}
			case mtproto.Predicate_inputReplyToStory:
				// TODO:
			}
		}

		if linkChatId > 0 {
			outMessage.Replies = mtproto.MakeTLMessageReplies(&mtproto.MessageReplies{
				Comments:       true,
				Replies:        0,
				RepliesPts:     0,
				RecentRepliers: nil,
				ChannelId:      mtproto.MakeFlagsInt64(linkChatId),
				MaxId:          nil,
				ReadMaxId:      nil,
			}).To_MessageReplies()
		}

		outMessage.Media, err = c.makeMediaByInputMedia(media.GetMedia())
		if err != nil {
			c.Logger.Errorf("messages.sendMultiMedia: %v", err)
			return nil, err
		}
		//outMessage, _ = c.fixMessageEntities(c.MD.UserId, peer, true, outMessage, func() bool {
		//	hasBot := c.MD.IsBot
		//	if !hasBot {
		//		//isBot, _ := c.svcCtx.Dao.UserClient.UserIsBot(c.ctx, &userpb.TLUserIsBot{
		//		//	Id: peer.PeerId,
		//		//})
		//		//hasBot = mtproto.FromBool(isBot)
		//	}
		//
		//	return hasBot
		//})
		outboxMultiMedia = append(outboxMultiMedia, msgpb.MakeTLOutboxMessage(&msgpb.OutboxMessage{
			NoWebpage:    true,
			Background:   in.Background,
			RandomId:     media.RandomId,
			Message:      outMessage,
			ScheduleDate: in.ScheduleDate,
		}).To_OutboxMessage())
	}

	rUpdate, err := c.svcCtx.Dao.MsgClient.MsgSendMessageV2(
		c.ctx,
		&msgpb.TLMsgSendMessageV2{
			UserId:    c.MD.UserId,
			AuthKeyId: c.MD.PermAuthKeyId,
			PeerType:  peer.PeerType,
			PeerId:    peer.PeerId,
			Message:   outboxMultiMedia,
		})

	if err != nil {
		c.Logger.Errorf("messages.sendMedia#c8f16791 - error: %v", err)
		return nil, err
	}

	if in.ClearDraft {
		ctx := contextx.ValueOnlyFrom(c.ctx)
		threading.GoSafe(func() {
			c.doClearDraft(ctx, c.MD.UserId, c.MD.PermAuthKeyId, peer)
		})
	}

	return rUpdate, nil
}
