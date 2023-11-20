import React from 'react';

import { SceneComponentProps, SceneGridItem, SceneObjectBase, SceneObjectRef, VizPanel } from '@grafana/scenes';
import { t } from 'app/core/internationalization';
import { ShareLibraryPanel } from 'app/features/dashboard/components/ShareModal/ShareLibraryPanel';
import { DashboardModel, PanelModel } from 'app/features/dashboard/state';

import { DashboardScene } from '../scene/DashboardScene';
import { PanelRepeaterGridItem } from '../scene/PanelRepeaterGridItem';
import { gridItemToPanel, transformSceneToSaveModel } from '../serialization/transformSceneToSaveModel';

import { SceneShareTabState } from './types';

export interface ShareLibraryPanelTabState extends SceneShareTabState {
  panelRef?: SceneObjectRef<VizPanel>;
  dashboardRef: SceneObjectRef<DashboardScene>;
}

export class ShareLibraryPanelTab extends SceneObjectBase<ShareLibraryPanelTabState> {
  static Component = ShareLibraryPanelTabRenderer;

  public getTabLabel() {
    return t('share-modal.tab-title.library-panel', 'Library panel');
  }
}

function ShareLibraryPanelTabRenderer({ model }: SceneComponentProps<ShareLibraryPanelTab>) {
  const { panelRef, dashboardRef, modalRef } = model.useState();

  if (!panelRef) {
    return null;
  }

  const vizPanel = panelRef.resolve();

  if (vizPanel.parent instanceof SceneGridItem || vizPanel.parent instanceof PanelRepeaterGridItem) {
    const panelJson = gridItemToPanel(vizPanel.parent);
    const panelModel = new PanelModel(panelJson);

    const dashboardJson = transformSceneToSaveModel(dashboardRef.resolve());
    const dashboardModel = new DashboardModel(dashboardJson);

    return (
      <ShareLibraryPanel
        dashboard={dashboardModel}
        panel={panelModel}
        onDismiss={() => {
          modalRef?.resolve().onDismiss();
        }}
      />
    );
  }

  return null;
}
