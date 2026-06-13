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

import React from 'react';
import { Tag } from '@douyinfe/semi-ui';

export const PRIORITY_OPTIONS = (t) => [
  { label: t('低'), value: 'low' },
  { label: t('中'), value: 'medium' },
  { label: t('高'), value: 'high' },
];

export const STATUS_OPTIONS = (t) => [
  { label: t('待处理'), value: 'open' },
  { label: t('处理中'), value: 'processing' },
  { label: t('已解决'), value: 'closed' },
];

export const renderPriorityTag = (priority, t) => {
  const map = {
    low: { color: 'grey', text: t('低') },
    medium: { color: 'blue', text: t('中') },
    high: { color: 'red', text: t('高') },
  };
  const item = map[priority] || map.medium;
  return (
    <Tag color={item.color} shape='circle' size='small'>
      {item.text}
    </Tag>
  );
};

export const renderStatusTag = (status, t) => {
  const map = {
    open: { color: 'orange', text: t('待处理') },
    processing: { color: 'blue', text: t('处理中') },
    closed: { color: 'green', text: t('已解决') },
  };
  const item = map[status] || map.open;
  return (
    <Tag color={item.color} shape='circle' size='small'>
      {item.text}
    </Tag>
  );
};
