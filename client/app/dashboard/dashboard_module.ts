//
// Copyright 2019 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS-IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
//
import {CommonModule} from '@angular/common';
import {HttpClientJsonpModule, HttpClientModule} from '@angular/common/http';
import {NgModule} from '@angular/core';
import {FormsModule} from '@angular/forms';
import {MatButtonModule} from '@angular/material/button';
import {MatDialogModule} from '@angular/material/dialog';
import {MatIconModule} from '@angular/material/icon';
import {MatInputModule} from '@angular/material/input';
import {MatProgressSpinnerModule} from '@angular/material/progress-spinner';
import {MatSidenavModule} from '@angular/material/sidenav';
import {MatSnackBarModule} from '@angular/material/snack-bar';
import {MatTooltipModule} from '@angular/material/tooltip';
import {BrowserModule} from '@angular/platform-browser';

import {HeatmapModule} from '../heatmap/heatmap_module';
import {SidebarModule} from '../sidebar/sidebar_module';

import {Dashboard} from './dashboard';
import {DashboardToolbar, DialogEditCollection} from './dashboard_toolbar';

@NgModule({
  declarations: [Dashboard, DashboardToolbar, DialogEditCollection],
  exports: [Dashboard, DashboardToolbar],
  imports: [
    BrowserModule,
    FormsModule,
    MatButtonModule,
    MatDialogModule,
    MatIconModule,
    MatInputModule,
    MatSnackBarModule,
    MatTooltipModule,
    HttpClientModule,
    HttpClientJsonpModule,
    HeatmapModule,
    MatProgressSpinnerModule,
    MatSidenavModule,
    SidebarModule,
    CommonModule,
  ],
  entryComponents: [DialogEditCollection]
})
export class DashboardModule {
}
