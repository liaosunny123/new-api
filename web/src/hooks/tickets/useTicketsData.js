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

import { useCallback, useEffect, useState } from 'react';
import { API, showError, showSuccess } from '../../helpers';

const PAGE_SIZE = 10;

/**
 * useTicketsData encapsulates ticket list/detail/create/reply logic.
 * When admin is true, it targets the admin endpoints and supports status/priority filters.
 */
export const useTicketsData = (admin = false) => {
  const base = admin ? '/api/admin/ticket' : '/api/ticket';

  const [tickets, setTickets] = useState([]);
  const [loading, setLoading] = useState(false);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [statusFilter, setStatusFilter] = useState('');
  const [priorityFilter, setPriorityFilter] = useState('');

  const [activeTicket, setActiveTicket] = useState(null);
  const [detailLoading, setDetailLoading] = useState(false);

  const loadTickets = useCallback(
    async (targetPage = page) => {
      setLoading(true);
      try {
        let url = `${base}/?p=${targetPage}&page_size=${PAGE_SIZE}`;
        if (admin) {
          if (statusFilter) url += `&status=${statusFilter}`;
          if (priorityFilter) url += `&priority=${priorityFilter}`;
        }
        const res = await API.get(url);
        const { success, message, data } = res.data;
        if (success) {
          setTickets(data.items || []);
          setTotal(data.total || 0);
          setPage(data.page || targetPage);
        } else {
          showError(message);
        }
      } catch (err) {
        showError(err.message);
      } finally {
        setLoading(false);
      }
    },
    [base, admin, page, statusFilter, priorityFilter],
  );

  useEffect(() => {
    loadTickets(1);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [admin, statusFilter, priorityFilter]);

  const openTicket = useCallback(
    async (id) => {
      setDetailLoading(true);
      try {
        const res = await API.get(`${base}/${id}`);
        const { success, message, data } = res.data;
        if (success) {
          setActiveTicket(data);
        } else {
          showError(message);
        }
      } catch (err) {
        showError(err.message);
      } finally {
        setDetailLoading(false);
      }
    },
    [base],
  );

  const closeTicket = useCallback(() => setActiveTicket(null), []);

  const createTicket = useCallback(
    async ({ title, content, priority }) => {
      const res = await API.post(`/api/ticket/`, { title, content, priority });
      const { success, message } = res.data;
      if (success) {
        showSuccess(message || 'OK');
        await loadTickets(1);
      } else {
        showError(message);
      }
      return success;
    },
    [loadTickets],
  );

  const replyTicket = useCallback(
    async (id, { content, status }) => {
      const payload = { content };
      if (admin && status) payload.status = status;
      const res = await API.post(`${base}/${id}/reply`, payload);
      const { success, message } = res.data;
      if (success) {
        await openTicket(id);
        await loadTickets(page);
      } else {
        showError(message);
      }
      return success;
    },
    [base, admin, openTicket, loadTickets, page],
  );

  const updateStatus = useCallback(
    async (id, status) => {
      const res = await API.put(`${base}/${id}/status`, { status });
      const { success, message } = res.data;
      if (success) {
        showSuccess(message || 'OK');
        await openTicket(id);
        await loadTickets(page);
      } else {
        showError(message);
      }
      return success;
    },
    [base, openTicket, loadTickets, page],
  );

  return {
    tickets,
    loading,
    page,
    total,
    pageSize: PAGE_SIZE,
    setPage: (p) => {
      setPage(p);
      loadTickets(p);
    },
    statusFilter,
    setStatusFilter,
    priorityFilter,
    setPriorityFilter,
    loadTickets,
    activeTicket,
    detailLoading,
    openTicket,
    closeTicket,
    createTicket,
    replyTicket,
    updateStatus,
  };
};

export default useTicketsData;
