/*
  openDox - Medical transcription platform
  Copyright (C) 2025  S2A Open Technologies gGmbH
  SPDX-FileCopyrightText: 2025 S2A Open Technologies gGmbH
  SPDX-License-Identifier: AGPL-3.0-or-later

  This program is free software: you can redistribute it and/or modify
  it under the terms of the GNU Affero General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.

  This program is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU Affero General Public License for more details.

  You should have received a copy of the GNU Affero General Public License
  along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import {useState} from 'react';
import {AlertTriangle} from 'lucide-react';
import {Trans, useTranslation} from 'react-i18next';
import {useAuth} from '@/contexts/AuthContext';
import {Button} from '@/components/ui/Button';
import './DisclaimerGate.css';

export const DisclaimerGate = () => {
    const {t} = useTranslation();
    const {acceptDisclaimer, logout} = useAuth();
    const [confirmed, setConfirmed] = useState(false);
    const [isSubmitting, setIsSubmitting] = useState(false);
    const [error, setError] = useState<string | null>(null);

    const handleAccept = async () => {
        setError(null);
        setIsSubmitting(true);
        const ok = await acceptDisclaimer();
        setIsSubmitting(false);
        if (!ok) {
            setError(t('disclaimer.failed'));
        }
    };

    return (
        <div className="disclaimer">
            {/* Drag-Region am oberen Rand */}
            <div className="disclaimer-dragbar"/>

            <div className="disclaimer-inner">
                <div className="disclaimer-alert">
                    <div className="disclaimer-alert-header">
                        <AlertTriangle size={16}/>
                        <span>{t('disclaimer.title')}</span>
                    </div>
                    <p>{t('disclaimer.p1')}</p>
                    <p>{t('disclaimer.p2')}</p>
                    <p><Trans i18nKey="disclaimer.p3" components={{code: <code/>}}/></p>
                </div>

                <label className="disclaimer-checkbox-row">
                    <input
                        type="checkbox"
                        checked={confirmed}
                        onChange={(e) => setConfirmed(e.target.checked)}
                        disabled={isSubmitting}
                    />
                    <span>{t('disclaimer.checkbox')}</span>
                </label>

                {error && <div className="disclaimer-error">{error}</div>}

                <div className="disclaimer-actions">
                    <Button variant="ghost" size="sm" onClick={() => logout()} disabled={isSubmitting}>
                        {t('common.logout')}
                    </Button>
                    <Button
                        variant="primary"
                        size="sm"
                        onClick={handleAccept}
                        disabled={!confirmed || isSubmitting}
                    >
                        {isSubmitting ? t('disclaimer.submitting') : t('common.confirm')}
                    </Button>
                </div>
            </div>
        </div>
    );
};

export default DisclaimerGate;
