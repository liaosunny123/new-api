/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React, { useState } from 'react';
import {
  Modal,
  Typography,
  Space,
  Tag,
  Avatar,
  Button,
  TextArea,
  Select,
  Spin,
  Divider,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { MarkdownRenderer } from '../common/markdown/MarkdownRenderer';
import {
  renderPriorityTag,
  renderStatusTag,
  STATUS_OPTIONS,
} from './ticketShared';
import { timestamp2string } from '../../helpers';

const { Text } = Typography;

const MessageBubble = ({ message, t }) => {
  const isAdmin = message.is_admin;
  return (
    <div
      className='flex'
      style={{
        justifyContent: isAdmin ? 'flex-start' : 'flex-end',
        marginBottom: 16,
      }}
    >
      <div
        style={{
          maxWidth: '80%',
          display: 'flex',
          gap: 8,
          flexDirection: isAdmin ? 'row' : 'row-reverse',
        }}
      >
        <Avatar size='small' color={isAdmin ? 'orange' : 'blue'}>
          {isAdmin ? t('管理员').slice(0, 1) : t('我')}
        </Avatar>
        <div>
          <div
            style={{
              display: 'flex',
              justifyContent: isAdmin ? 'flex-start' : 'flex-end',
              gap: 6,
              marginBottom: 2,
            }}
          >
            <Text size='small' type='tertiary'>
              {isAdmin ? t('管理员') : t('我')}
            </Text>
            <Text size='small' type='tertiary'>
              {timestamp2string(message.created_time)}
            </Text>
          </div>
          <div
            style={{
              padding: '8px 12px',
              borderRadius: 10,
              background: isAdmin
                ? 'var(--semi-color-fill-0)'
                : 'var(--semi-color-primary-light-default)',
            }}
          >
            <MarkdownRenderer content={message.content} />
          </div>
        </div>
      </div>
    </div>
  );
};

const TicketDetailModal = ({
  visible,
  ticket,
  loading,
  admin = false,
  onClose,
  onReply,
  onUpdateStatus,
}) => {
  const { t } = useTranslation();
  const [reply, setReply] = useState('');
  const [replyStatus, setReplyStatus] = useState('');
  const [submitting, setSubmitting] = useState(false);

  const handleReply = async () => {
    if (!reply.trim()) return;
    setSubmitting(true);
    const ok = await onReply(ticket.id, {
      content: reply,
      status: admin ? replyStatus : undefined,
    });
    setSubmitting(false);
    if (ok) {
      setReply('');
      setReplyStatus('');
    }
  };

  return (
    <Modal
      title={
        ticket ? (
          <Space>
            <span>{`#${ticket.id}`}</span>
            <span>{ticket.title}</span>
            {renderStatusTag(ticket.status, t)}
            {renderPriorityTag(ticket.priority, t)}
          </Space>
        ) : (
          t('工单详情')
        )
      }
      visible={visible}
      onCancel={onClose}
      footer={null}
      width={760}
    >
      {loading || !ticket ? (
        <div style={{ textAlign: 'center', padding: 40 }}>
          <Spin />
        </div>
      ) : (
        <div>
          {admin && ticket.username ? (
            <Text type='tertiary' size='small'>
              {t('提交用户')}: {ticket.username}
            </Text>
          ) : null}
          <div
            style={{
              maxHeight: 380,
              overflowY: 'auto',
              padding: '12px 4px',
              marginTop: 8,
            }}
          >
            {(ticket.messages || []).map((m) => (
              <MessageBubble key={m.id} message={m} t={t} />
            ))}
          </div>

          <Divider margin='12px' />

          {ticket.status === 'closed' && !admin ? (
            <Text type='tertiary'>
              {t('该工单已解决，如有需要可继续回复以重新打开。')}
            </Text>
          ) : null}

          <TextArea
            placeholder={t('输入回复内容，支持 Markdown')}
            value={reply}
            onChange={setReply}
            autosize={{ minRows: 3, maxRows: 8 }}
          />

          <div
            style={{
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
              marginTop: 12,
              gap: 8,
            }}
          >
            <div>
              {admin && (
                <Space>
                  <Text type='tertiary' size='small'>
                    {t('回复后状态')}
                  </Text>
                  <Select
                    placeholder={t('保持不变')}
                    optionList={STATUS_OPTIONS(t)}
                    value={replyStatus}
                    onChange={setReplyStatus}
                    style={{ width: 140 }}
                    showClear
                  />
                </Space>
              )}
            </div>
            <Space>
              {admin && (
                <Button
                  type='tertiary'
                  onClick={() => onUpdateStatus(ticket.id, 'closed')}
                  disabled={ticket.status === 'closed'}
                >
                  {t('标记为已解决')}
                </Button>
              )}
              <Button
                theme='solid'
                loading={submitting}
                onClick={handleReply}
                disabled={!reply.trim()}
              >
                {t('发送回复')}
              </Button>
            </Space>
          </div>
        </div>
      )}
    </Modal>
  );
};

export default TicketDetailModal;
