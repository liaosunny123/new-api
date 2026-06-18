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

import React, { useEffect, useState, useRef } from 'react';
import {
  Banner,
  Button,
  Form,
  Row,
  Col,
  Typography,
  Spin,
} from '@douyinfe/semi-ui';
const { Text } = Typography;
import {
  API,
  removeTrailingSlash,
  showError,
  showSuccess,
} from '../../../helpers';
import { useTranslation } from 'react-i18next';

export default function SettingsPaymentGateway(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    StripeEnabled: false,
    StripeApiSecret: '',
    StripeWebhookSecret: '',
    StripeUnitPrice: 8.0,
    StripeMinTopUp: 1,
    StripePromotionCodesEnabled: false,
    StripeCurrency: 'usd',
    StripeExchangeRate: 1,
    StripeMinAmount: 0,
    StripeMaxAmount: 0,
    StripeFeeEnabled: false,
    StripeFeePercent: 3.4,
    StripeFeeFixed: 2.35,
  });
  const [originInputs, setOriginInputs] = useState({});
  const formApiRef = useRef(null);

  useEffect(() => {
    if (props.options && formApiRef.current) {
      const num = (v, d) => (v !== undefined && v !== null && v !== '' ? parseFloat(v) : d);
      const currentInputs = {
        StripeEnabled: props.options.StripeEnabled === true || props.options.StripeEnabled === 'true',
        StripeApiSecret: props.options.StripeApiSecret || '',
        StripeWebhookSecret: props.options.StripeWebhookSecret || '',
        StripeUnitPrice: num(props.options.StripeUnitPrice, 8.0),
        StripeMinTopUp: num(props.options.StripeMinTopUp, 1),
        StripePromotionCodesEnabled:
          props.options.StripePromotionCodesEnabled === true ||
          props.options.StripePromotionCodesEnabled === 'true',
        StripeCurrency: props.options.StripeCurrency || 'usd',
        StripeExchangeRate: num(props.options.StripeExchangeRate, 1),
        StripeMinAmount: num(props.options.StripeMinAmount, 0),
        StripeMaxAmount: num(props.options.StripeMaxAmount, 0),
        StripeFeeEnabled:
          props.options.StripeFeeEnabled === true ||
          props.options.StripeFeeEnabled === 'true',
        StripeFeePercent: num(props.options.StripeFeePercent, 3.4),
        StripeFeeFixed: num(props.options.StripeFeeFixed, 2.35),
      };
      setInputs(currentInputs);
      setOriginInputs({ ...currentInputs });
      formApiRef.current.setValues(currentInputs);
    }
  }, [props.options]);

  const handleFormChange = (values) => {
    setInputs(values);
  };

  const submitStripeSetting = async () => {
    if (props.options.ServerAddress === '') {
      showError(t('请先填写服务器地址'));
      return;
    }

    setLoading(true);
    try {
      const options = [];

      // 密钥类：仅在填写时提交（敏感信息不回显）
      if (inputs.StripeApiSecret && inputs.StripeApiSecret !== '') {
        options.push({ key: 'StripeApiSecret', value: inputs.StripeApiSecret });
      }
      if (inputs.StripeWebhookSecret && inputs.StripeWebhookSecret !== '') {
        options.push({ key: 'StripeWebhookSecret', value: inputs.StripeWebhookSecret });
      }

      // 开关
      options.push({ key: 'StripeEnabled', value: inputs.StripeEnabled ? 'true' : 'false' });
      options.push({ key: 'StripeFeeEnabled', value: inputs.StripeFeeEnabled ? 'true' : 'false' });
      options.push({
        key: 'StripePromotionCodesEnabled',
        value: inputs.StripePromotionCodesEnabled ? 'true' : 'false',
      });

      // 文本 / 数值
      options.push({ key: 'StripeCurrency', value: (inputs.StripeCurrency || 'usd').toLowerCase() });
      const numKeys = [
        'StripeUnitPrice',
        'StripeMinTopUp',
        'StripeExchangeRate',
        'StripeMinAmount',
        'StripeMaxAmount',
        'StripeFeePercent',
        'StripeFeeFixed',
      ];
      numKeys.forEach((k) => {
        if (inputs[k] !== undefined && inputs[k] !== null) {
          options.push({ key: k, value: inputs[k].toString() });
        }
      });

      const requestQueue = options.map((opt) =>
        API.put('/api/option/', { key: opt.key, value: opt.value }),
      );

      const results = await Promise.all(requestQueue);

      const errorResults = results.filter((res) => !res.data.success);
      if (errorResults.length > 0) {
        errorResults.forEach((res) => {
          showError(res.data.message);
        });
      } else {
        showSuccess(t('更新成功'));
        setOriginInputs({ ...inputs });
        props.refresh?.();
      }
    } catch (error) {
      showError(t('更新失败'));
    }
    setLoading(false);
  };

  return (
    <Spin spinning={loading}>
      <Form
        initValues={inputs}
        onValueChange={handleFormChange}
        getFormApi={(api) => (formApiRef.current = api)}
      >
        <Form.Section text={t('Stripe 设置')}>
          <Text>
            Stripe 密钥、Webhook 等设置请
            <a
              href='https://dashboard.stripe.com/developers'
              target='_blank'
              rel='noreferrer'
            >
              点击此处
            </a>
            进行设置，最好先在
            <a
              href='https://dashboard.stripe.com/test/developers'
              target='_blank'
              rel='noreferrer'
            >
              测试环境
            </a>
            进行测试。
            <br />
          </Text>
          <Banner
            type='info'
            description={`Webhook 填：${props.options.ServerAddress ? removeTrailingSlash(props.options.ServerAddress) : t('网站地址')}/api/stripe/webhook`}
          />
          <Banner
            type='warning'
            description={`需要包含事件：checkout.session.completed 和 checkout.session.expired`}
          />
          <Banner
            type='info'
            description={t(
              '使用受限密钥(rk_)时，最小权限：Checkout Sessions 写、Customers 写、PaymentIntents 读、Charges 读。',
            )}
          />
          <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Switch
                field='StripeEnabled'
                size='default'
                checkedText='｜'
                uncheckedText='〇'
                label={t('开启 Stripe 支付')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='StripeApiSecret'
                label={t('API 密钥')}
                placeholder={t('sk_xxx 或 rk_xxx 的 Stripe 密钥，敏感信息不显示')}
                type='password'
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='StripeWebhookSecret'
                label={t('Webhook 签名密钥')}
                placeholder={t('whsec_xxx 的 Webhook 签名密钥，敏感信息不显示')}
                type='password'
              />
            </Col>
          </Row>
          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.InputNumber
                field='StripeUnitPrice'
                precision={2}
                label={t('充值价格（x元/美金）')}
                placeholder={t('例如：7，就是7元/美金')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.InputNumber
                field='StripeMinTopUp'
                label={t('最低充值数量')}
                placeholder={t('例如：2')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Switch
                field='StripePromotionCodesEnabled'
                size='default'
                checkedText='｜'
                uncheckedText='〇'
                label={t('允许在 Stripe 支付中输入促销码')}
              />
            </Col>
          </Row>

          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Input
                field='StripeCurrency'
                label={t('扣款货币（如 usd、hkd、cny）')}
                placeholder='usd'
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.InputNumber
                field='StripeExchangeRate'
                precision={4}
                label={t('汇率：1 CNY = ? 扣款货币')}
                placeholder={t('例如 1 CNY = 0.14 USD 就填 0.14；货币为 cny 填 1')}
              />
            </Col>
          </Row>
          <Banner
            type='info'
            description={t(
              '充值金额按人民币(CNY)计算，结账时乘以上述汇率换算为扣款货币向用户收取。',
            )}
          />

          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.InputNumber
                field='StripeMinAmount'
                precision={2}
                label={t('最低支付金额（CNY，0=不限）')}
                placeholder='50'
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.InputNumber
                field='StripeMaxAmount'
                precision={2}
                label={t('最高支付金额（CNY，0=不限）')}
                placeholder='1000'
              />
            </Col>
          </Row>
          <Banner
            type='info'
            description={t(
              '当用户充值金额超出 [最低, 最高] 区间时，点击 Stripe 支付会提示其使用其他方式或联系管理员。',
            )}
          />

          <Row
            gutter={{ xs: 8, sm: 16, md: 24, lg: 24, xl: 24, xxl: 24 }}
            style={{ marginTop: 16 }}
          >
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.Switch
                field='StripeFeeEnabled'
                size='default'
                checkedText='｜'
                uncheckedText='〇'
                label={t('向客户加收 Stripe 处理费')}
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.InputNumber
                field='StripeFeePercent'
                precision={2}
                label={t('处理费率(%)')}
                placeholder='3.4'
              />
            </Col>
            <Col xs={24} sm={24} md={8} lg={8} xl={8}>
              <Form.InputNumber
                field='StripeFeeFixed'
                precision={2}
                label={t('固定处理费（扣款货币）')}
                placeholder='2.35'
              />
            </Col>
          </Row>
          <Banner
            type='info'
            description={t(
              '开启后收银台会增加一行“Stripe 处理费”，按 (金额+固定费)/(1-费率) 反算，使你到手≈原始金额。',
            )}
          />

          <Button style={{ marginTop: 16 }} onClick={submitStripeSetting}>
            {t('更新 Stripe 设置')}
          </Button>
        </Form.Section>
      </Form>
    </Spin>
  );
}
