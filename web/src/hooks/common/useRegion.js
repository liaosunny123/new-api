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

const CACHE_KEY = 'region_info';

// In-memory promise so multiple hook consumers share a single request per page load.
let regionPromise = null;

const fetchRegion = async () => {
  // Session cache to avoid repeated lookups while navigating.
  try {
    const cached = sessionStorage.getItem(CACHE_KEY);
    if (cached) {
      return JSON.parse(cached);
    }
  } catch (e) {
    // ignore
  }
  if (!regionPromise) {
    regionPromise = API.get('/api/region')
      .then((res) => {
        const { success, data } = res.data;
        const info = success
          ? { country: data.country || '', isMainland: !!data.is_mainland }
          : { country: '', isMainland: false };
        try {
          sessionStorage.setItem(CACHE_KEY, JSON.stringify(info));
        } catch (e) {
          // ignore
        }
        return info;
      })
      .catch(() => ({ country: '', isMainland: false }));
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
