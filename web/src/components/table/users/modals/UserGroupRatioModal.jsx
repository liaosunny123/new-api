import React, { useEffect, useState, useCallback } from 'react';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../../../helpers';
import {
  Banner,
  Button,
  Card,
  Empty,
  InputNumber,
  Select,
  SideSheet,
  Space,
  Spin,
  Switch,
  Typography,
} from '@douyinfe/semi-ui';
import { IconSave, IconPlus, IconDelete } from '@douyinfe/semi-icons';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';

const { Text, Title } = Typography;

// 仅用于在编辑期间为每一行生成稳定 key，避免使用 Date/Math.random
let rowSeq = 0;
const nextRowId = () => {
  rowSeq += 1;
  return `gr_${rowSeq}`;
};

const UserGroupRatioModal = ({ visible, onCancel, userId, username }) => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [enabled, setEnabled] = useState(false);
  const [rows, setRows] = useState([]); // [{ _id, group, ratio }]
  const [groupOptions, setGroupOptions] = useState([]);

  const fetchData = useCallback(async () => {
    if (!userId) return;
    try {
      const [overridesRes, groupsRes] = await Promise.all([
        API.get(`/api/user/${userId}/group-ratio-overrides`),
        API.get(`/api/group/`),
      ]);
      if (overridesRes.data.success) {
        const data = overridesRes.data.data || {};
        setEnabled(!!data.enabled);
        const overrides = data.overrides || {};
        setRows(
          Object.entries(overrides).map(([group, ratio]) => ({
            _id: nextRowId(),
            group,
            ratio,
          })),
        );
      }
      if (groupsRes.data.success) {
        const list = groupsRes.data.data || [];
        setGroupOptions(list.map((g) => ({ label: g, value: g })));
      }
    } catch (e) {
      console.error('fetch group ratio data error:', e);
    }
  }, [userId]);

  useEffect(() => {
    if (visible && userId) {
      setLoading(true);
      fetchData().finally(() => setLoading(false));
    }
  }, [visible, userId, fetchData]);

  const addRow = () => {
    setRows((prev) => [...prev, { _id: nextRowId(), group: '', ratio: 1 }]);
  };

  const removeRow = (id) => {
    setRows((prev) => prev.filter((r) => r._id !== id));
  };

  const updateRow = (id, field, value) => {
    setRows((prev) =>
      prev.map((r) => (r._id === id ? { ...r, [field]: value } : r)),
    );
  };

  const handleSave = async () => {
    // 组装为 map，校验重复/非法分组
    const overrides = {};
    for (const r of rows) {
      const group = (r.group || '').trim();
      if (!group) {
        showError(t('存在未选择分组的行，请先选择分组或删除该行'));
        return;
      }
      if (overrides[group] !== undefined) {
        showError(t('分组 {{group}} 重复，请合并', { group }));
        return;
      }
      const ratio = r.ratio == null ? 0 : r.ratio;
      if (ratio < 0) {
        showError(t('分组 {{group}} 的倍率不能为负数', { group }));
        return;
      }
      overrides[group] = ratio;
    }

    setSaving(true);
    try {
      const res = await API.put(`/api/user/${userId}/group-ratio-overrides`, {
        enabled,
        overrides,
      });
      if (res.data.success) {
        showSuccess(t('保存成功'));
        onCancel?.();
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e.message);
    }
    setSaving(false);
  };

  return (
    <SideSheet
      placement='right'
      title={
        <Space>
          <Title heading={4} className='m-0'>
            {t('分组价格设置')} - {username}
          </Title>
        </Space>
      }
      visible={visible}
      width={isMobile ? '100%' : 560}
      onCancel={onCancel}
      footer={
        <div className='flex justify-end'>
          <Space>
            <Button
              theme='solid'
              icon={<IconSave />}
              onClick={handleSave}
              loading={saving}
            >
              {t('保存')}
            </Button>
          </Space>
        </div>
      }
    >
      <Spin spinning={loading}>
        <div className='p-2 space-y-3'>
          <Banner
            type='info'
            closeIcon={null}
            description={t(
              '为该用户单独设置某些分组的倍率，优先级高于系统分组倍率。仅在下方总开关开启时生效。',
            )}
          />

          <Card className='!rounded-xl shadow-sm border-0'>
            <div className='flex items-center justify-between'>
              <div>
                <Text strong style={{ fontSize: 15 }}>
                  {t('启用分组价格设置')}
                </Text>
                <div>
                  <Text size='small' type='tertiary'>
                    {t('关闭后所有分组倍率覆盖均不生效')}
                  </Text>
                </div>
              </div>
              <Switch checked={enabled} onChange={(v) => setEnabled(v)} />
            </div>
          </Card>

          <Card
            className='!rounded-xl shadow-sm border-0'
            title={
              <Text strong style={{ fontSize: 15 }}>
                {t('分组倍率')}
              </Text>
            }
          >
            {rows.length === 0 ? (
              <Empty
                title={t('暂无分组倍率')}
                description={t('点击下方按钮添加分组并设置倍率')}
              />
            ) : (
              <div className='space-y-2'>
                {rows.map((r) => (
                  <div key={r._id} className='flex items-center gap-2'>
                    <Select
                      size='small'
                      filter
                      allowCreate
                      placeholder={t('选择分组')}
                      value={r.group || undefined}
                      optionList={groupOptions}
                      onChange={(v) => updateRow(r._id, 'group', v)}
                      style={{ flex: 1 }}
                      position='bottomLeft'
                      disabled={!enabled}
                    />
                    <InputNumber
                      size='small'
                      min={0}
                      step={0.1}
                      value={r.ratio}
                      style={{ width: 120 }}
                      onChange={(v) => updateRow(r._id, 'ratio', v)}
                      disabled={!enabled}
                    />
                    <Button
                      size='small'
                      type='danger'
                      theme='borderless'
                      icon={<IconDelete />}
                      onClick={() => removeRow(r._id)}
                      disabled={!enabled}
                    />
                  </div>
                ))}
              </div>
            )}
            <div className='mt-3'>
              <Button
                size='small'
                icon={<IconPlus />}
                onClick={addRow}
                disabled={!enabled}
              >
                {t('添加分组')}
              </Button>
            </div>
          </Card>
        </div>
      </Spin>
    </SideSheet>
  );
};

export default UserGroupRatioModal;
