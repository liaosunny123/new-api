import React, { useEffect, useState, useCallback, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../../../helpers';
import {
  Button,
  Card,
  InputNumber,
  Progress,
  SideSheet,
  Space,
  Spin,
  Switch,
  Typography,
  Empty,
} from '@douyinfe/semi-ui';
import { IconSave, IconRefresh } from '@douyinfe/semi-icons';
import { useIsMobile } from '../../../../hooks/common/useIsMobile';

const { Text, Title } = Typography;

const UserRateLimitModal = ({ visible, onCancel, userId, username }) => {
  const { t } = useTranslation();
  const isMobile = useIsMobile();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [usage, setUsage] = useState({});
  const [overrides, setOverrides] = useState({});
  const timerRef = useRef(null);

  const fetchData = useCallback(async () => {
    if (!userId) return;
    try {
      const [usageRes, overridesRes] = await Promise.all([
        API.get(`/api/user/${userId}/rate-limit-usage`),
        API.get(`/api/user/${userId}/rate-limit-overrides`),
      ]);
      if (usageRes.data.success) {
        setUsage(usageRes.data.data || {});
      }
      if (overridesRes.data.success) {
        setOverrides(overridesRes.data.data || {});
      }
    } catch (e) {
      console.error('fetch rate limit data error:', e);
    }
  }, [userId]);

  const fetchUsageOnly = useCallback(async () => {
    if (!userId) return;
    try {
      const res = await API.get(`/api/user/${userId}/rate-limit-usage`);
      if (res.data.success) {
        setUsage(res.data.data || {});
      }
    } catch (e) {
      console.error('fetch usage error:', e);
    }
  }, [userId]);

  useEffect(() => {
    if (visible && userId) {
      setLoading(true);
      fetchData().finally(() => setLoading(false));
      // Auto-refresh usage every 5 seconds
      timerRef.current = setInterval(fetchUsageOnly, 5000);
    }
    return () => {
      if (timerRef.current) {
        clearInterval(timerRef.current);
        timerRef.current = null;
      }
    };
  }, [visible, userId, fetchData, fetchUsageOnly]);

  const handleSave = async () => {
    setSaving(true);
    try {
      // Clean up overrides: remove entries where both rpm and concurrency are null
      const cleanOverrides = {};
      for (const [group, override] of Object.entries(overrides)) {
        if (override.rpm != null || override.concurrency != null) {
          cleanOverrides[group] = override;
        }
      }
      const res = await API.put(`/api/user/${userId}/rate-limit-overrides`, {
        overrides: cleanOverrides,
      });
      if (res.data.success) {
        showSuccess(t('保存成功'));
        setOverrides(cleanOverrides);
        fetchUsageOnly();
      } else {
        showError(res.data.message);
      }
    } catch (e) {
      showError(e.message);
    }
    setSaving(false);
  };

  const updateOverride = (group, field, value) => {
    setOverrides((prev) => ({
      ...prev,
      [group]: {
        ...prev[group],
        [field]: value,
      },
    }));
  };

  const toggleUseDefault = (group, field, useDefault) => {
    if (useDefault) {
      updateOverride(group, field, null);
    } else {
      // Set to current limit as starting point
      const currentLimit = usage[group]?.[field === 'rpm' ? 'rpm_limit' : 'concurrency_limit'] || 0;
      updateOverride(group, field, currentLimit);
    }
  };

  const getProgressPercent = (current, limit) => {
    if (!limit || limit <= 0) return 0;
    return Math.min(Math.round((current / limit) * 100), 100);
  };

  const getProgressStatus = (percent) => {
    if (percent >= 90) return 'exception';
    if (percent >= 70) return 'warning';
    return 'success';
  };

  const groupNames = Object.keys(usage);

  return (
    <SideSheet
      placement='right'
      title={
        <Space>
          <Title heading={4} className='m-0'>
            {t('限流设置')} - {username}
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
              {t('保存覆盖设置')}
            </Button>
            <Button
              theme='light'
              icon={<IconRefresh />}
              onClick={() => {
                setLoading(true);
                fetchData().finally(() => setLoading(false));
              }}
            >
              {t('刷新')}
            </Button>
          </Space>
        </div>
      }
    >
      <Spin spinning={loading}>
        <div className='p-2 space-y-3'>
          {groupNames.length === 0 ? (
            <Empty
              title={t('暂无限流配置')}
              description={t('该用户没有配置RPM/并发限制的分组')}
            />
          ) : (
            groupNames.map((group) => {
              const u = usage[group] || {};
              const o = overrides[group] || {};
              const rpmPercent = getProgressPercent(u.rpm_current || 0, u.rpm_limit || 0);
              const concPercent = getProgressPercent(
                u.concurrency_current || 0,
                u.concurrency_limit || 0,
              );

              return (
                <Card
                  key={group}
                  className='!rounded-xl shadow-sm border-0'
                  title={
                    <Text strong style={{ fontSize: 15 }}>
                      {group}
                    </Text>
                  }
                >
                  {/* RPM */}
                  {u.rpm_limit > 0 && (
                    <div style={{ marginBottom: 16 }}>
                      <div className='flex items-center justify-between mb-1'>
                        <Text size='small' strong>
                          RPM ({t('每分钟请求数')})
                        </Text>
                        <Text size='small' type='tertiary'>
                          {u.rpm_current || 0} / {u.rpm_limit || 0}
                        </Text>
                      </div>
                      <Progress
                        percent={rpmPercent}
                        showInfo={false}
                        size='large'
                        stroke={
                          rpmPercent >= 90
                            ? 'var(--semi-color-danger)'
                            : rpmPercent >= 70
                              ? 'var(--semi-color-warning)'
                              : 'var(--semi-color-success)'
                        }
                      />
                      <div className='flex items-center gap-2 mt-2'>
                        <Text size='small'>{t('自定义覆盖')}:</Text>
                        <Switch
                          size='small'
                          checked={o.rpm != null}
                          onChange={(checked) =>
                            toggleUseDefault(group, 'rpm', !checked)
                          }
                        />
                        {o.rpm != null && (
                          <InputNumber
                            size='small'
                            min={0}
                            step={1}
                            value={o.rpm}
                            style={{ width: 100 }}
                            onChange={(v) => updateOverride(group, 'rpm', v ?? 0)}
                          />
                        )}
                      </div>
                    </div>
                  )}

                  {/* Concurrency */}
                  {u.concurrency_limit > 0 && (
                    <div>
                      <div className='flex items-center justify-between mb-1'>
                        <Text size='small' strong>
                          {t('并发数')}
                        </Text>
                        <Text size='small' type='tertiary'>
                          {u.concurrency_current || 0} / {u.concurrency_limit || 0}
                        </Text>
                      </div>
                      <Progress
                        percent={concPercent}
                        showInfo={false}
                        size='large'
                        stroke={
                          concPercent >= 90
                            ? 'var(--semi-color-danger)'
                            : concPercent >= 70
                              ? 'var(--semi-color-warning)'
                              : 'var(--semi-color-success)'
                        }
                      />
                      <div className='flex items-center gap-2 mt-2'>
                        <Text size='small'>{t('自定义覆盖')}:</Text>
                        <Switch
                          size='small'
                          checked={o.concurrency != null}
                          onChange={(checked) =>
                            toggleUseDefault(group, 'concurrency', !checked)
                          }
                        />
                        {o.concurrency != null && (
                          <InputNumber
                            size='small'
                            min={0}
                            step={1}
                            value={o.concurrency}
                            style={{ width: 100 }}
                            onChange={(v) =>
                              updateOverride(group, 'concurrency', v ?? 0)
                            }
                          />
                        )}
                      </div>
                    </div>
                  )}
                </Card>
              );
            })
          )}
        </div>
      </Spin>
    </SideSheet>
  );
};

export default UserRateLimitModal;
