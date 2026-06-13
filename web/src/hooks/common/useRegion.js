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

import { useEffect, useState } from 'react';
import { API } from '../../helpers';

// In-memory promise so multiple hook consumers share a single request per page
// load. Intentionally NOT persisted to sessionStorage/localStorage: a stale
// "not mainland" result (e.g. cached before the GeoIP DB was deployed) must not
// survive across reloads/redeploys. A full page reload re-fetches.
let regionPromise = null;

// Clean up the legacy persistent cache key (previously stored a possibly-stale
// region result). Safe no-op if absent.
try {
  sessionStorage.removeItem('region_info');
} catch (e) {
  // ignore
}

const fetchRegion = async () => {
  if (!regionPromise) {
    regionPromise = API.get('/api/region')
      .then((res) => {
        const { success, data } = res.data;
        return success
          ? { country: data.country || '', isMainland: !!data.is_mainland }
          : { country: '', isMainland: false };
      })
      .catch(() => {
        // Reset so a transient failure can be retried on the next consumer.
        regionPromise = null;
        return { country: '', isMainland: false };
      });
  }
  return regionPromise;
};

/**
 * useRegion resolves the visitor's region via the backend GeoIP endpoint.
 * Returns { country, isMainland, loading }. Defaults to non-mainland on any failure
 * so groups remain visible (consistent with the backend default-allow behavior).
 */
export const useRegion = () => {
  const [state, setState] = useState({
    country: '',
    isMainland: false,
    loading: true,
  });

  useEffect(() => {
    let mounted = true;
    fetchRegion().then((info) => {
      if (mounted) {
        setState({ ...info, loading: false });
      }
    });
    return () => {
      mounted = false;
    };
  }, []);

  return state;
};

export default useRegion;
