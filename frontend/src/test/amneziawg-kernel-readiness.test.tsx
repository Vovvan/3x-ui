import { Form } from 'antd';
import { FormProvider, useForm } from 'react-hook-form';
import { describe, expect, it, vi } from 'vitest';
import { waitFor } from '@testing-library/react';

import AmneziawgFields from '@/pages/inbounds/form/protocols/amneziawg';
import { HttpUtil, Msg } from '@/utils';
import { makeTestQueryClient, renderWithProviders } from './test-utils';

interface FormValues {
  settings: { server: { kernelMode: boolean } };
}

function Harness({ kernelMode }: { kernelMode: boolean }) {
  const form = useForm<FormValues>({
    defaultValues: { settings: { server: { kernelMode } } },
  });
  return (
    <FormProvider {...form}>
      <Form>
        <AmneziawgFields
          awgPubKey="public-key"
          regenInboundAwg={() => {}}
          regenInboundAwgObfuscation={() => {}}
        />
      </Form>
    </FormProvider>
  );
}

const unavailable = new Msg(true, '', {
  ready: false,
  blockers: [{ code: 'tools', detail: 'awg' }],
  installCommand: 'sudo x-ui install-awg-kernel',
});

describe('AmneziaWG kernel readiness in the inbound form', () => {
  it('shows an informational warning while kernel mode is off', async () => {
    vi.spyOn(HttpUtil, 'get').mockResolvedValue(unavailable);

    const { container, getByRole } = renderWithProviders(<Harness kernelMode={false} />, {
      queryClient: makeTestQueryClient(),
    });

    await waitFor(() => expect(getByRole('alert')).toBeTruthy());
    expect(container.querySelector('.ant-alert-info')).toBeTruthy();
    expect(container.querySelector('.ant-alert-warning')).toBeNull();
  });

  it('shows a warning when an enabled local kernel inbound is unavailable', async () => {
    vi.spyOn(HttpUtil, 'get').mockResolvedValue(unavailable);

    const { container, getByRole } = renderWithProviders(<Harness kernelMode />, {
      queryClient: makeTestQueryClient(),
    });

    await waitFor(() => expect(getByRole('alert')).toBeTruthy());
    expect(container.querySelector('.ant-alert-warning')).toBeTruthy();
  });
});
