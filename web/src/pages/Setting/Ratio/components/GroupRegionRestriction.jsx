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

import React, { useMemo } from 'react';
import { Switch, Typography, Empty, Table } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';

const { Text } = Typography;

function parseJSON(str) {
  if (!str || !str.trim()) return {};
  try {
    return JSON.parse(str);
  } catch {
    return {};
  }
}

/**
 * GroupRegionRestriction lets admins toggle, per group, whether mainland China
 * users are allowed to use it. Default (no entry) = allowed.
 *
 * value: JSON string mapping groupName -> bool
 * onChange(jsonString)
 * groupNames: list of all known group names (from GroupRatio)
 */
const GroupRegionRestriction = ({ value, onChange, groupNames = [] }) => {
  const { t } = useTranslation();
  const map = useMemo(() => parseJSON(value), [value]);

  const handleToggle = (groupName, allow) => {
    const next = { ...map, [groupName]: allow };
    onChange(JSON.stringify(next));
  };

  const dataSource = groupNames.map((name) => ({
    key: name,
    group: name,
    allow: map[name] !== false, // default allowed
  }));

  const columns = [
    {
      title: t('分组'),
      dataIndex: 'group',
      render: (text) => <Text strong>{text}</Text>,
    },
    {
      title: t('允许大陆用户使用'),
      dataIndex: 'allow',
      width: 200,
      render: (allow, record) => (
        <Switch
          checked={allow}
          onChange={(checked) => handleToggle(record.group, checked)}
        />
      ),
    },
  ];

  if (!groupNames.length) {
    return <Empty description={t('暂无分组')} />;
  }

  return (
    <div>
      <Text
        type='tertiary'
        size='small'
        style={{ display: 'block', marginBottom: 12 }}
      >
        {t(
          '关闭后，中国大陆地区的用户在前端将无法看到并选择该分组（令牌管理、模型广场），但后端仍可正常调用',
        )}
      </Text>
      <Table
        columns={columns}
        dataSource={dataSource}
        pagination={false}
        size='small'
      />
    </div>
  );
};

export default GroupRegionRestriction;
