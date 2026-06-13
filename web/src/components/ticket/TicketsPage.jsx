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
import { Table, Button, Typography, Space, Select } from '@douyinfe/semi-ui';
import { IconPlus } from '@douyinfe/semi-icons';
import { useTranslation } from 'react-i18next';
import CardPro from '../common/ui/CardPro';
import { useIsMobile } from '../../hooks/common/useIsMobile';
import { createCardProPagination, timestamp2string } from '../../helpers';
import { useTicketsData } from '../../hooks/tickets/useTicketsData';
import {
  renderPriorityTag,
  renderStatusTag,
  STATUS_OPTIONS,
  PRIORITY_OPTIONS,
} from './ticketShared';
import CreateTicketModal from './CreateTicketModal';
import TicketDetailModal from './TicketDetailModal';

const { Title, Text } = Typography;

const TicketsPage = ({ admin = false }) => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const data = useTicketsData(admin);
  const [showCreate, setShowCreate] = useState(false);

  const columns = [
    {
      title: 'ID',
      dataIndex: 'id',
      width: 70,
    },
    {
      title: t('主题'),
      dataIndex: 'title',
      render: (text) => (
        <Text ellipsis={{ showTooltip: true }} style={{ maxWidth: 260 }}>
          {text}
        </Text>
      ),
    },
    ...(admin
      ? [
          {
            title: t('提交用户'),
            dataIndex: 'username',
            width: 140,
            render: (text) => text || '-',
          },
        ]
      : []),
    {
      title: t('优先级'),
      dataIndex: 'priority',
      width: 90,
      render: (p) => renderPriorityTag(p, t),
    },
    {
      title: t('状态'),
      dataIndex: 'status',
      width: 100,
      render: (s) => renderStatusTag(s, t),
    },
    {
      title: t('更新时间'),
      dataIndex: 'updated_time',
      width: 170,
      render: (ts) => timestamp2string(ts),
    },
    {
      title: t('操作'),
      dataIndex: 'op',
      width: 100,
      render: (_, record) => (
        <Button
          theme='light'
          type='primary'
          size='small'
          onClick={() => data.openTicket(record.id)}
        >
          {admin ? t('处理') : t('查看')}
        </Button>
      ),
    },
  ];

  const descriptionArea = (
    <div className='flex items-center justify-between w-full flex-wrap gap-2'>
      <div>
        <Title heading={6} style={{ margin: 0 }}>
          {admin ? t('工单管理') : t('我的工单')}
        </Title>
        <Text type='tertiary' size='small'>
          {admin
            ? t('查看并回复用户提交的工单')
            : t('提交问题反馈，每天最多可提交 10 个工单')}
        </Text>
      </div>
      {!admin && (
        <Button
          theme='solid'
          icon={<IconPlus />}
          onClick={() => setShowCreate(true)}
        >
          {t('提交工单')}
        </Button>
      )}
    </div>
  );

  const searchArea = admin ? (
    <Space wrap>
      <Select
        placeholder={t('全部状态')}
        optionList={STATUS_OPTIONS(t)}
        value={data.statusFilter || undefined}
        onChange={(v) => data.setStatusFilter(v || '')}
        style={{ width: 150 }}
        showClear
      />
      <Select
        placeholder={t('全部优先级')}
        optionList={PRIORITY_OPTIONS(t)}
        value={data.priorityFilter || undefined}
        onChange={(v) => data.setPriorityFilter(v || '')}
        style={{ width: 150 }}
        showClear
      />
    </Space>
  ) : null;

  const paginationArea = createCardProPagination({
    currentPage: data.page,
    pageSize: data.pageSize,
    total: data.total,
    onPageChange: (p) => data.setPage(p),
    isMobile,
    showSizeChanger: false,
    t,
  });

  return (
    <>
      <CardPro
        type='type1'
        descriptionArea={descriptionArea}
        searchArea={searchArea}
        paginationArea={paginationArea}
        t={t}
      >
        <Table
          columns={columns}
          dataSource={data.tickets}
          loading={data.loading}
          pagination={false}
          rowKey='id'
          size='middle'
        />
      </CardPro>

      <CreateTicketModal
        visible={showCreate}
        onClose={() => setShowCreate(false)}
        onSubmit={data.createTicket}
      />

      <TicketDetailModal
        visible={!!data.activeTicket}
        ticket={data.activeTicket}
        loading={data.detailLoading}
        admin={admin}
        onClose={data.closeTicket}
        onReply={data.replyTicket}
        onUpdateStatus={data.updateStatus}
      />
    </>
  );
};

export default TicketsPage;
