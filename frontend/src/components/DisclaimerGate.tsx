/*
  openDox - Medical transcription platform
  Copyright (C) 2025  Christopher Abanilla <christopher.abanilla@gmx.de>

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
import {useAuth} from '@/contexts/AuthContext';
import {Button} from '@/components/ui/Button';
import './DisclaimerGate.css';

export const DisclaimerGate = () => {
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
            setError('Bestätigung fehlgeschlagen – bitte erneut versuchen.');
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
                        <span>Kein Medizinprodukt</span>
                    </div>
                    <p>
                        openDox ist kein zugelassenes Medizinprodukt gemäß Medizinproduktegesetz (MPG)
                        oder vergleichbaren Regularien (z. B. EU-MDR, FDA). Es dient ausschließlich der
                        Unterstützung bei der Dokumentation.
                    </p>
                    <p>
                        Alle maschinell erstellten Transkriptionen und Zusammenfassungen müssen von
                        qualifiziertem Fachpersonal überprüft werden, bevor sie verwendet oder in eine
                        Patientenakte übernommen werden. openDox ersetzt keine ärztliche Diagnose,
                        Beratung oder Therapieempfehlung.
                    </p>
                    <p>
                        Wer diese Anwendung betreibt, trägt die Verantwortung dafür, dass der Einsatz mit
                        den in der eigenen Jurisdiktion geltenden Vorschriften vereinbar ist. Details siehe{' '}
                        <code>DISCLAIMER.md</code> im Projekt-Repository.
                    </p>
                </div>

                <label className="disclaimer-checkbox-row">
                    <input
                        type="checkbox"
                        checked={confirmed}
                        onChange={(e) => setConfirmed(e.target.checked)}
                        disabled={isSubmitting}
                    />
                    <span>
                        Ich habe den Hinweis gelesen und verstanden und bestätige, dass ich für die
                        regelkonforme Nutzung dieser Anwendung verantwortlich bin.
                    </span>
                </label>

                {error && <div className="disclaimer-error">{error}</div>}

                <div className="disclaimer-actions">
                    <Button variant="ghost" size="sm" onClick={() => logout()} disabled={isSubmitting}>
                        Abmelden
                    </Button>
                    <Button
                        variant="primary"
                        size="sm"
                        onClick={handleAccept}
                        disabled={!confirmed || isSubmitting}
                    >
                        {isSubmitting ? 'Bestätige...' : 'Bestätigen'}
                    </Button>
                </div>
            </div>
        </div>
    );
};

export default DisclaimerGate;
