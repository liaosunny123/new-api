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
  Form,
  Typography,
  Tabs,
  TabPane,
  Empty,
} from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import { MarkdownRenderer } from '../common/markdown/MarkdownRenderer';
import { PRIORITY_OPTIONS } from './ticketShared';

const { Text } = Typography;

const CreateTicketModal = ({ visible, onClose, onSubmit }) => {
  const { t } = useTranslation();
  const [title, setTitle] = useState('');
  const [content, setContent] = useState('');
  const [priority, setPriority] = useState('medium');
  const [submitting, setSubmitting] = useState(false);

  const reset = () => {
    setTitle('');
    setContent('');
    setPriority('medium');
  };

  const handleOk = async () => {
    setSubmitting(true);
    const ok = await onSubmit({ title, content, priority });
    setSubmitting(false);
    if (ok) {
      reset();
      onClose();
    }
  };

  return (
    <Modal
      title={t('提交工单')}
      visible={visible}
      onOk={handleOk}
      onCancel={() => {
        reset();
        onClose();
      }}
      okButtonProps={{ loading: submitting }}
      okText={t('提交')}
      cancelText={t('取消')}
      width={680}
    >
      <Form labelPosition='top'>
        <Form.Input
          field='title'
          label={t('主题')}
          placeholder={t('请输入工单主题')}
          value={title}
          onChange={setTitle}
          maxLength={255}
          showClear
        />
        <Form.Select
          field='priority'
          label={t('优先级')}
          optionList={PRIORITY_OPTIONS(t)}
          value={priority}
          onChange={setPriority}
          style={{ width: 160 }}
        />
        <div style={{ marginTop: 8 }}>
          <Text strong>{t('内容')}</Text>
          <Text type='tertiary' size='small' style={{ marginLeft: 8 }}>
            {t('支持 Markdown 格式')}
          </Text>
        </div>
        <Tabs type='line' style={{ marginTop: 4 }}>
          <TabPane tab={t('编辑')} itemKey='edit'>
            <Form.TextArea
              field='content'
              noLabel
              placeholder={t('请输入工单内容，支持 Markdown')}
              value={content}
              onChange={setContent}
              autosize={{ minRows: 8, maxRows: 18 }}
            />
          </TabPane>
          <TabPane tab={t('预览')} itemKey='preview'>
            <div
              style={{
                minHeight: 180,
                padding: 12,
                border: '1px solid var(--semi-color-border)',
                borderRadius: 8,
              }}
            >
              {content ? (
                <MarkdownRenderer content={content} />
              ) : (
                <Empty description={t('暂无内容')} />
              )}
            </div>
          </TabPane>
        </Tabs>
      </Form>
    </Modal>
  );
};

export default CreateTicketModal;
