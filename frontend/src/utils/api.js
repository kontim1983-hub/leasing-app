import axios from 'axios';

export const API_URL = process.env.REACT_APP_API_URL || 'http://localhost:8080';

// Общие вспомогательные функции для работы с таблицами
export const isColumnChanged = (record, column) => record.changed_columns?.includes(column);

export const getCellClass = (record, column) => {
    if (record.is_new) return 'new-row';
    if (isColumnChanged(record, column)) return 'changed-cell';
    return '';
};

// API функции для Tab1
export const fetchRecords = async () => {
    const res = await axios.get(`${API_URL}/api/records`);
    return res.data || [];
};

export const fetchFiles = async () => {
    const res = await fetch(`${API_URL}/api/files`);
    if (!res.ok) return [];
    const data = await res.json();
    return Array.isArray(data) ? data : [];
};

export const uploadFile = async (file) => {
    const formData = new FormData();
    formData.append('file', file);

    const res = await axios.post(`${API_URL}/api/upload`, formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
    });

    return res.data || [];
};

export const clearChangedColumns = async () => {
    const response = await fetch(`${API_URL}/api/clear-changed-columns`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
    });
    return await response.json();
};

export const deleteAllRecords = async () => {
    const response = await fetch(`${API_URL}/api/delete-all-records`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ confirm: "delete" }),
    });

    if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`Ошибка сервера: ${response.status} — ${errorText}`);
    }

    return await response.json();
};

// API функции для Tab2
export const fetchRecordsV2 = async () => {
    const res = await axios.get(`${API_URL}/api/v2/records`);
    return res.data || [];
};
export const fetchRecordsV3 = async () => {
    const res = await axios.get(`${API_URL}/api/v3/records`);
    return res.data || [];
};

export const fetchFilesV2 = async () => {
    const res = await fetch(`${API_URL}/api/v2/files`);
    if (!res.ok) return [];
    const data = await res.json();
    return Array.isArray(data) ? data : [];
};
export const fetchFilesV3 = async () => {
    const res = await fetch(`${API_URL}/api/v3/files`);
    if (!res.ok) return [];
    const data = await res.json();
    return Array.isArray(data) ? data : [];
};

export const uploadFileV2 = async (file) => {
    const formData = new FormData();
    formData.append('file', file);

    const res = await axios.post(`${API_URL}/api/v2/upload`, formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
    });

    return res.data.records || [];
};
export const uploadFileV3 = async (file) => {
    const formData = new FormData();
    formData.append('file', file);

    const res = await axios.post(`${API_URL}/api/v3/upload`, formData, {
        headers: { 'Content-Type': 'multipart/form-data' },
    });

    return res.data.records || [];
};

export const clearChangedColumnsV2 = async () => {
    const response = await fetch(`${API_URL}/api/v2/clear-changed-columns`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
    });
    return await response.json();
};
export const clearChangedColumnsV3 = async () => {
    const response = await fetch(`${API_URL}/api/v3/clear-changed-columns`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
    });
    return await response.json();
};

export const deleteAllRecordsV2 = async () => {
    const response = await fetch(`${API_URL}/api/v2/delete-all-records`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ confirm: "delete" }),
    });

    if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`Ошибка сервера: ${response.status} — ${errorText}`);
    }

    return await response.json();
};
export const deleteAllRecordsV3 = async () => {
    const response = await fetch(`${API_URL}/api/v3/delete-all-records`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ confirm: "delete" }),
    });

    if (!response.ok) {
        const errorText = await response.text();
        throw new Error(`Ошибка сервера: ${response.status} — ${errorText}`);
    }

    return await response.json();
};
export const exportExcelV3 = async () => {
    const response = await fetch(`${API_URL}/api/v3/export`);
    if (!response.ok) throw new Error('Ошибка экспорта');

    const blob = await response.blob();
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'leasing_records_v2.xlsx';
    document.body.appendChild(a);
    a.click();
    window.URL.revokeObjectURL(url);
    document.body.removeChild(a);
};
export const exportExcelV2 = async () => {
    const response = await fetch(`${API_URL}/api/v2/export`);
    if (!response.ok) throw new Error('Ошибка экспорта');

    const blob = await response.blob();
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'leasing_records_v2.xlsx';
    document.body.appendChild(a);
    a.click();
    window.URL.revokeObjectURL(url);
    document.body.removeChild(a);
};

export const exportExcel = async () => {
    const response = await fetch(`${API_URL}/api/export`);
    if (!response.ok) throw new Error('Ошибка экспорта');

    const blob = await response.blob();
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = 'leasing_records.xlsx';
    document.body.appendChild(a);
    a.click();
    window.URL.revokeObjectURL(url);
    document.body.removeChild(a);
};