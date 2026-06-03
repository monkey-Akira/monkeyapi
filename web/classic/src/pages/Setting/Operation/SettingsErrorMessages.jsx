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

import React, { useEffect, useRef, useState } from 'react';
import { Button, Col, Form, Row, Spin, Typography } from '@douyinfe/semi-ui';
import { useTranslation } from 'react-i18next';
import {
  API,
  compareObjects,
  showError,
  showSuccess,
  showWarning,
} from '../../../helpers';

const defaultMappings = {
  insufficient_user_quota: '\u989d\u5ea6\u4e0d\u8db3\uff0c\u8bf7\u5145\u503c\u540e\u518d\u8bd5\u3002',
  pre_consume_token_quota_failed:
    '\u989d\u5ea6\u9884\u6263\u5931\u8d25\uff0c\u8bf7\u7a0d\u540e\u91cd\u8bd5\u3002',
  'channel:no_available_key':
    '\u5f53\u524d\u6a21\u578b\u6682\u4e0d\u53ef\u7528\uff0c\u8bf7\u7a0d\u540e\u518d\u8bd5\u3002',
  'channel:invalid_key':
    '\u5f53\u524d\u6a21\u578b\u6682\u4e0d\u53ef\u7528\uff0c\u8bf7\u8054\u7cfb\u7ba1\u7406\u5458\u5904\u7406\u3002',
  'channel:model_mapped_error':
    '\u6a21\u578b\u914d\u7f6e\u5f02\u5e38\uff0c\u8bf7\u8054\u7cfb\u7ba1\u7406\u5458\u5904\u7406\u3002',
  do_request_failed:
    '\u8fde\u63a5\u4e0a\u6e38\u670d\u52a1\u5931\u8d25\uff0c\u8bf7\u7a0d\u540e\u91cd\u8bd5\u3002',
  bad_response_status_code:
    '\u4e0a\u6e38\u670d\u52a1\u54cd\u5e94\u5f02\u5e38\uff0c\u8bf7\u7a0d\u540e\u91cd\u8bd5\u3002',
  bad_response_body:
    '\u4e0a\u6e38\u670d\u52a1\u8fd4\u56de\u5f02\u5e38\uff0c\u8bf7\u7a0d\u540e\u91cd\u8bd5\u3002',
  model_not_found: '\u5f53\u524d\u6a21\u578b\u6682\u4e0d\u53ef\u7528\u3002',
  prompt_blocked:
    '\u8bf7\u6c42\u5185\u5bb9\u672a\u901a\u8fc7\u5b89\u5168\u68c0\u67e5\uff0c\u8bf7\u8c03\u6574\u540e\u91cd\u8bd5\u3002',
  sensitive_words_detected:
    '\u8bf7\u6c42\u5185\u5bb9\u5305\u542b\u654f\u611f\u8bcd\uff0c\u8bf7\u8c03\u6574\u540e\u91cd\u8bd5\u3002',
  invalid_request:
    '\u8bf7\u6c42\u53c2\u6570\u6709\u8bef\uff0c\u8bf7\u68c0\u67e5\u540e\u91cd\u8bd5\u3002',
};

export default function SettingsErrorMessages(props) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [inputs, setInputs] = useState({
    'error_message_setting.enabled': false,
    'error_message_setting.mappings': '{}',
  });
  const refForm = useRef();
  const [inputsRow, setInputsRow] = useState(inputs);

  function handleFieldChange(fieldName) {
    return (value) => {
      setInputs((current) => ({ ...current, [fieldName]: value }));
    };
  }

  function validateMappings(value) {
    try {
      const parsed = JSON.parse(value || '{}');
      if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
        return false;
      }
      return Object.entries(parsed).every(
        ([code, message]) =>
          code.trim() !== '' &&
          typeof message === 'string' &&
          message.trim() !== '',
      );
    } catch {
      return false;
    }
  }

  function useDefaultMappings() {
    const mappings = JSON.stringify(defaultMappings, null, 2);
    setInputs((current) => ({
      ...current,
      'error_message_setting.mappings': mappings,
    }));
    refForm.current.setValue('error_message_setting.mappings', mappings);
  }

  function onSubmit() {
    if (!validateMappings(inputs['error_message_setting.mappings'])) {
      showError(t('Error message mappings must be a JSON object.'));
      return;
    }
    const updateArray = compareObjects(inputs, inputsRow);
    if (!updateArray.length) return showWarning(t('No changes to save'));
    const requestQueue = updateArray.map((item) => {
      const value =
        typeof inputs[item.key] === 'boolean'
          ? String(inputs[item.key])
          : String(inputs[item.key]);
      return API.put('/api/option/', {
        key: item.key,
        value,
      });
    });
    setLoading(true);
    Promise.all(requestQueue)
      .then((res) => {
        if (requestQueue.length === 1) {
          if (res.includes(undefined)) return;
        } else if (requestQueue.length > 1) {
          if (res.includes(undefined))
            return showError(t('Some settings failed to save, please retry'));
        }
        showSuccess(t('Saved successfully'));
        props.refresh();
      })
      .catch(() => {
        showError(t('Save failed, please retry'));
      })
      .finally(() => {
        setLoading(false);
      });
  }

  useEffect(() => {
    const currentInputs = {};
    for (let key in props.options) {
      if (Object.keys(inputs).includes(key)) {
        currentInputs[key] = props.options[key];
      }
    }
    setInputs(currentInputs);
    setInputsRow(structuredClone(currentInputs));
    refForm.current.setValues(currentInputs);
  }, [props.options]);

  return (
    <>
      <Spin spinning={loading}>
        <Form
          values={inputs}
          getFormApi={(formAPI) => (refForm.current = formAPI)}
          style={{ marginBottom: 15 }}
        >
          <Form.Section text={t('Error Response Messages')}>
            <Typography.Text
              type='tertiary'
              style={{ marginBottom: 16, display: 'block' }}
            >
              {t(
                'Replace returned error.message by error code. Logs and error codes are kept unchanged.',
              )}
            </Typography.Text>
            <Row gutter={16}>
              <Col xs={24} sm={12} md={8} lg={8} xl={8}>
                <Form.Switch
                  field={'error_message_setting.enabled'}
                  label={t('Enable custom error messages')}
                  size='default'
                  checkedText='ON'
                  uncheckedText='OFF'
                  onChange={handleFieldChange('error_message_setting.enabled')}
                />
              </Col>
              <Col span={24}>
                <Form.TextArea
                  field={'error_message_setting.mappings'}
                  label={t('Error code mappings')}
                  placeholder='{"bad_response_status_code":"Upstream service is unavailable. Please try again later."}'
                  autosize
                  onChange={handleFieldChange('error_message_setting.mappings')}
                />
              </Col>
            </Row>
            <Row>
              <Button size='default' onClick={useDefaultMappings}>
                {t('Use default mappings')}
              </Button>
              <Button size='default' onClick={onSubmit} style={{ marginLeft: 8 }}>
                {t('Save error messages')}
              </Button>
            </Row>
          </Form.Section>
        </Form>
      </Spin>
    </>
  );
}
