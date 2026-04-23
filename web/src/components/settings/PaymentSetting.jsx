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

import React, { useEffect, useState } from 'react';
import { Card, Spin } from '@douyinfe/semi-ui';
import SettingsGeneralPayment from '../../pages/Setting/Payment/SettingsGeneralPayment';
import SettingsPaymentGateway from '../../pages/Setting/Payment/SettingsPaymentGateway';
import SettingsPaymentGatewayStripe from '../../pages/Setting/Payment/SettingsPaymentGatewayStripe';
import SettingsPaymentGatewayCreem from '../../pages/Setting/Payment/SettingsPaymentGatewayCreem';
import SettingsPaymentGatewayWaffo from '../../pages/Setting/Payment/SettingsPaymentGatewayWaffo';
import { API, showError, toBoolean } from '../../helpers';
import { useTranslation } from 'react-i18next';

const PaymentSetting = () => {
  const { t } = useTranslation();
  let [inputs, setInputs] = useState({
    ServerAddress: '',
    PayAddress: '',
    EpayId: '',
    EpayKey: '',
    Price: 7.3,
    MinTopUp: 1,
    TopupGroupRatio: '',
    CustomCallbackAddress: '',
    PayMethods: '',
    AmountOptions: '',
    AmountDiscount: '',

    SingleTopUpLimitEnabled: false,
    SingleTopUpLimitAmount: 0,
    SingleTopUpLimitMessage: '',
    DailyTopUpLimitEnabled: false,
    DailyTopUpLimitAmount: 0,
    DailyTopUpLimitMessage: '',

    StripeApiSecret: '',
    StripeWebhookSecret: '',
    StripePriceId: '',
    StripeUnitPrice: 8.0,
    StripeMinTopUp: 1,
    StripePromotionCodesEnabled: false,
  });

  let [loading, setLoading] = useState(false);

  const getOptions = async () => {
    const res = await API.get('/api/option/');
    const { success, message, data } = res.data;
    if (success) {
      let newInputs = {};
      data.forEach((item) => {
        switch (item.key) {
          case 'TopupGroupRatio':
            try {
              newInputs[item.key] = JSON.stringify(
                JSON.parse(item.value),
                null,
                2,
              );
            } catch (error) {
              newInputs[item.key] = item.value;
            }
            break;
          case 'payment_setting.amount_options':
            try {
              newInputs['AmountOptions'] = JSON.stringify(
                JSON.parse(item.value),
                null,
                2,
              );
            } catch (error) {
              newInputs['AmountOptions'] = item.value;
            }
            break;
          case 'payment_setting.amount_discount':
            try {
              newInputs['AmountDiscount'] = JSON.stringify(
                JSON.parse(item.value),
                null,
                2,
              );
            } catch (error) {
              newInputs['AmountDiscount'] = item.value;
            }
            break;
          case 'payment_setting.single_topup_limit_enabled':
            newInputs['SingleTopUpLimitEnabled'] = item.value === 'true';
            break;
          case 'payment_setting.single_topup_limit_amount':
            newInputs['SingleTopUpLimitAmount'] = parseFloat(item.value) || 0;
            break;
          case 'payment_setting.single_topup_limit_message':
            newInputs['SingleTopUpLimitMessage'] = item.value;
            break;
          case 'payment_setting.daily_topup_limit_enabled':
            newInputs['DailyTopUpLimitEnabled'] = item.value === 'true';
            break;
          case 'payment_setting.daily_topup_limit_amount':
            newInputs['DailyTopUpLimitAmount'] = parseFloat(item.value) || 0;
            break;
          case 'payment_setting.daily_topup_limit_message':
            newInputs['DailyTopUpLimitMessage'] = item.value;
            break;
          case 'Price':
          case 'MinTopUp':
          case 'StripeUnitPrice':
          case 'StripeMinTopUp':
            newInputs[item.key] = parseFloat(item.value);
            break;
          default:
            if (item.key.endsWith('Enabled')) {
              newInputs[item.key] = toBoolean(item.value);
            } else {
              newInputs[item.key] = item.value;
            }
            break;
        }
      });

      setInputs(newInputs);
    } else {
      showError(t(message));
    }
  };

  async function onRefresh() {
    try {
      setLoading(true);
      await getOptions();
    } catch (error) {
      showError(t('刷新失败'));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    onRefresh();
  }, []);

  return (
    <>
      <Spin spinning={loading} size='large'>
        <Card style={{ marginTop: '10px' }}>
          <SettingsGeneralPayment options={inputs} refresh={onRefresh} />
        </Card>
        <Card style={{ marginTop: '10px' }}>
          <SettingsPaymentGateway options={inputs} refresh={onRefresh} />
        </Card>
        <Card style={{ marginTop: '10px' }}>
          <SettingsPaymentGatewayStripe options={inputs} refresh={onRefresh} />
        </Card>
        <Card style={{ marginTop: '10px' }}>
          <SettingsPaymentGatewayCreem options={inputs} refresh={onRefresh} />
        </Card>
        <Card style={{ marginTop: '10px' }}>
          <SettingsPaymentGatewayWaffo options={inputs} refresh={onRefresh} />
        </Card>
      </Spin>
    </>
  );
};

export default PaymentSetting;
