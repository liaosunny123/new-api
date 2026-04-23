import React, { useEffect, useState, useCallback, useRef } from 'react';
import { Card, Progress, Empty, Tag, Spin, Typography } from '@douyinfe/semi-ui';
import { Gauge } from 'lucide-react';
import { API } from '../../helpers';
import { useTranslation } from 'react-i18next';
import {
  IllustrationConstruction,
  IllustrationConstructionDark,
} from '@douyinfe/semi-illustrations';
import ScrollableContainer from '../common/ui/ScrollableContainer';

const { Text } = Typography;

const getPercent = (current, limit) => {
  if (!limit || limit <= 0) return 0;
  return Math.min(Math.round((current / limit) * 100), 100);
};

const getColor = (percent) => {
  if (percent >= 90) return 'var(--semi-color-danger)';
  if (percent >= 70) return 'var(--semi-color-warning)';
  return 'var(--semi-color-success)';
};

const GroupCard = ({ group, data, t }) => {
  const rpmPct = getPercent(data.rpm_current || 0, data.rpm_limit || 0);
  const concPct = getPercent(
    data.concurrency_current || 0,
    data.concurrency_limit || 0,
  );

  return (
    <div
      style={{
        border: '1px solid var(--semi-color-border)',
        borderRadius: 10,
        padding: '10px 14px',
        background: 'var(--semi-color-bg-1)',
        minWidth: 0,
      }}
    >
      <Tag size='small' color='blue' style={{ marginBottom: 8 }}>
        {group}
      </Tag>

      {data.rpm_limit > 0 && (
        <div style={{ marginBottom: data.concurrency_limit > 0 ? 8 : 0 }}>
          <div className='flex items-center justify-between'>
            <Text size='small'>RPM</Text>
            <Text size='small' type='tertiary'>
              {data.rpm_current || 0}/{data.rpm_limit}
            </Text>
          </div>
          <Progress
            percent={rpmPct}
            showInfo={false}
            size='small'
            stroke={getColor(rpmPct)}
          />
        </div>
      )}

      {data.concurrency_limit > 0 && (
        <div>
          <div className='flex items-center justify-between'>
            <Text size='small'>{t('并发')}</Text>
            <Text size='small' type='tertiary'>
              {data.concurrency_current || 0}/{data.concurrency_limit}
            </Text>
          </div>
          <Progress
            percent={concPct}
            showInfo={false}
            size='small'
            stroke={getColor(concPct)}
          />
        </div>
      )}
    </div>
  );
};

const RateLimitPanel = ({ CARD_PROPS, ILLUSTRATION_SIZE }) => {
  const { t } = useTranslation();
  const [usage, setUsage] = useState({});
  const [loading, setLoading] = useState(true);
  const timerRef = useRef(null);

  const fetchUsage = useCallback(async () => {
    try {
      const res = await API.get('/api/user/self/rate-limit-usage');
      if (res.data.success) {
        setUsage(res.data.data || {});
      }
    } catch (e) {
      // silent
    }
  }, []);

  useEffect(() => {
    fetchUsage().finally(() => setLoading(false));
    timerRef.current = setInterval(fetchUsage, 5000);
    return () => {
      if (timerRef.current) clearInterval(timerRef.current);
    };
  }, [fetchUsage]);

  const groups = Object.keys(usage);

  if (!loading && groups.length === 0) {
    return null;
  }

  return (
    <Card
      {...CARD_PROPS}
      className='shadow-sm !rounded-2xl'
      title={
        <div className='flex items-center gap-2'>
          <Gauge size={16} />
          {t('RPM / 并发限制')}
          {groups.length > 0 && (
            <Tag size='small' color='white' shape='circle'>
              {groups.length}
            </Tag>
          )}
        </div>
      }
      bodyStyle={{ padding: 0 }}
    >
      <Spin spinning={loading} size='small'>
        {groups.length === 0 ? (
          <div className='flex justify-center items-center py-6'>
            <Empty
              image={
                <IllustrationConstruction style={ILLUSTRATION_SIZE} />
              }
              darkModeImage={
                <IllustrationConstructionDark style={ILLUSTRATION_SIZE} />
              }
              title={t('暂无限流配置')}
              description={t('当前无 RPM/并发限制')}
            />
          </div>
        ) : (
          <ScrollableContainer maxHeight='13rem'>
            <div
              style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fill, minmax(180px, 1fr))',
                gap: 10,
                padding: '12px 16px',
              }}
            >
              {groups.map((group) => (
                <GroupCard
                  key={group}
                  group={group}
                  data={usage[group]}
                  t={t}
                />
              ))}
            </div>
          </ScrollableContainer>
        )}
      </Spin>
    </Card>
  );
};

export default RateLimitPanel;
