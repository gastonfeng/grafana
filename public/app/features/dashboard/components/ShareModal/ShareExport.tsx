import { saveAs } from 'file-saver';
import React, { PureComponent } from 'react';

import { reportInteraction } from '@grafana/runtime';
import { Button, Field, Modal, Switch } from '@grafana/ui';
import { appEvents } from 'app/core/core';
import { t, Trans } from 'app/core/internationalization';
import { DashboardExporter } from 'app/features/dashboard/components/DashExportModal';
import { ShowModalReactEvent } from 'app/types/events';

import { ViewJsonModal } from './ViewJsonModal';
import { ShareModalTabProps } from './types';

interface Props extends ShareModalTabProps {}

interface State {
  shareExternally: boolean;
}

export class ShareExport extends PureComponent<Props, State> {
  private exporter: DashboardExporter;

  constructor(props: Props) {
    super(props);
    this.state = {
      shareExternally: false,
    };

    this.exporter = new DashboardExporter();
  }

  onShareExternallyChange = () => {
    this.setState({
      shareExternally: !this.state.shareExternally,
    });
  };

  onSaveAsFile = () => {
    const { dashboard } = this.props;
    const { shareExternally } = this.state;

    reportInteraction('dashboards_sharing_export_save_json_clicked', { externally: shareExternally });

    if (shareExternally) {
      this.exporter.makeExportable(dashboard).then((dashboardJson) => {
        this.openSaveAsDialog(dashboardJson);
      });
    } else {
      this.openSaveAsDialog(dashboard.getSaveModelClone());
    }
  };

  onViewJson = () => {
    const { dashboard } = this.props;
    const { shareExternally } = this.state;

    reportInteraction('dashboards_sharing_export_view_json_clicked', { externally: shareExternally });

    if (shareExternally) {
      this.exporter.makeExportable(dashboard).then((dashboardJson) => {
        this.openJsonModal(dashboardJson);
      });
    } else {
      this.openJsonModal(dashboard.getSaveModelClone());
    }
  };

  openSaveAsDialog = (dash: any) => {
    const dashboardJsonPretty = JSON.stringify(dash, null, 2);
    const blob = new Blob([dashboardJsonPretty], {
      type: 'application/json;charset=utf-8',
    });
    const time = new Date().getTime();
    saveAs(blob, `${dash.title}-${time}.json`);
  };

  openJsonModal = (clone: object) => {
    appEvents.publish(
      new ShowModalReactEvent({
        props: {
          json: JSON.stringify(clone, null, 2),
        },
        component: ViewJsonModal,
      })
    );

    this.props.onDismiss?.();
  };

  render() {
    const { onDismiss } = this.props;
    const { shareExternally } = this.state;

    const exportExternallyTranslation = t('share-modal.export.share-externally-label', `Export for sharing externally`);

    return (
      <>
        <p className="share-modal-info-text">
          <Trans i18nKey="share-modal.export.info-text">Export this dashboard.</Trans>
        </p>
        <Field label={exportExternallyTranslation}>
          <Switch id="share-externally-toggle" value={shareExternally} onChange={this.onShareExternallyChange} />
        </Field>
        <Modal.ButtonRow>
          <Button variant="secondary" onClick={onDismiss} fill="outline">
            <Trans i18nKey="share-modal.export.cancel-button">Cancel</Trans>
          </Button>
          <Button variant="secondary" onClick={this.onViewJson}>
            <Trans i18nKey="share-modal.export.view-button">View JSON</Trans>
          </Button>
          <Button variant="primary" onClick={this.onSaveAsFile}>
            <Trans i18nKey="share-modal.export.save-button">Save to file</Trans>
          </Button>
        </Modal.ButtonRow>
      </>
    );
  }
}
