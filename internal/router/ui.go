package router

import "github.com/gin-gonic/gin"

func serveWebUI(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(200, kgUI)
}

const kgUI = `<!DOCTYPE html>
<html lang="zh-CN">
<head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1.0">
<title>知识图谱构建器</title>
<script src="https://cdn.jsdelivr.net/npm/echarts@5.5.0/dist/echarts.min.js"></script>
<style>
:root{--bg:#0f172a;--card:#1e293b;--border:#334155;--text:#e2e8f0;--muted:#94a3b8;--accent:#8b5cf6}
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:-apple-system,BlinkMacSystemFont,'Microsoft YaHei',sans-serif;background:var(--bg);color:var(--text);min-height:100vh}
.header{background:var(--card);border-bottom:1px solid var(--border);padding:12px 24px;display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:12px}
.header h1{font-size:18px}
.btn{padding:8px 18px;border-radius:6px;border:none;cursor:pointer;font-size:13px;font-weight:500}
.btn-primary{background:var(--accent);color:#fff}
input{background:var(--bg);border:1px solid var(--border);color:var(--text);padding:8px 12px;border-radius:6px;font-size:13px;outline:none}
.container{display:grid;grid-template-columns:300px 1fr;height:calc(100vh - 60px)}
@media(max-width:800px){.container{grid-template-columns:1fr;grid-template-rows:auto 1fr}}
.sidebar{background:var(--card);border-right:1px solid var(--border);padding:16px;overflow-y:auto}
.sidebar h3{font-size:14px;margin-bottom:12px;color:var(--accent)}
.upload-zone{border:2px dashed var(--border);border-radius:10px;padding:20px;text-align:center;cursor:pointer;margin-bottom:12px}
.upload-zone:hover{border-color:var(--accent)}
.entity-item,.relation-item{padding:8px 10px;border:1px solid var(--border);border-radius:6px;margin-bottom:6px;font-size:12px;cursor:pointer;transition:all .2s}
.entity-item:hover,.relation-item:hover{border-color:var(--accent)}
.entity-item .type-badge{display:inline-block;padding:1px 6px;border-radius:4px;font-size:10px;margin-right:6px}
#graphChart{width:100%;height:100%}
.toast-msg{padding:10px 18px;border-radius:8px;font-size:12px;margin-bottom:8px;position:fixed;top:20px;right:20px;z-index:9999}
.toast-success{background:#065f46;color:#6ee7b7}.toast-error{background:#7f1d1d;color:#fca5a5}
.spinner{display:inline-block;width:16px;height:16px;border:2px solid var(--border);border-top-color:var(--accent);border-radius:50%;animation:spin .6s linear infinite}
@keyframes spin{to{transform:rotate(360deg)}}
</style></head>
<body>
<div class="header">
  <h1>🧠 档案知识图谱构建器</h1>
  <div style="display:flex;gap:8px;align-items:center">
    <input id="searchInput" placeholder="搜索实体（人名/机构/合同号）" style="width:200px" onkeydown="if(event.key==='Enter')searchEntity()">
    <button class="btn btn-primary" onclick="searchEntity()">搜索</button>
  </div>
</div>
<div class="container">
  <div class="sidebar">
    <div class="upload-zone" id="dropZone" onclick="document.getElementById('fileInput').click()">
      <p style="font-size:28px">📄</p><p>上传文档</p>
      <p style="font-size:11px;color:var(--muted);margin-top:4px">PDF/DOCX/TXT → 自动提取实体和关系</p>
    </div>
    <input type="file" id="fileInput" accept=".pdf,.docx,.txt" style="display:none" onchange="uploadDoc()">
    <div id="uploadStatus" style="font-size:12px;color:var(--muted);margin-bottom:12px"></div>
    <h3>实体列表</h3><div id="entityList" style="max-height:300px;overflow-y:auto"><p style="font-size:12px;color:var(--muted)">上传文档后自动提取</p></div>
    <h3 style="margin-top:16px">关系列表</h3><div id="relationList" style="max-height:200px;overflow-y:auto"><p style="font-size:12px;color:var(--muted)">抽取的关系将显示在此</p></div>
  </div>
  <div id="graphChart"></div>
</div>
<div id="toast"></div>

<script>
var bp=window.location.pathname.replace(/\/+$/,'');var API=(bp===''||bp==='/')?'/api':bp+'/api';
var typeColors={'person':'#3b82f6','org':'#22c55e','contract':'#f59e0b','date':'#8b5cf6','money':'#ef4444','location':'#06b6d4'};
var typeLabels={'person':'人物','org':'机构','contract':'合同','date':'日期','money':'金额','location':'地点'};
var entityGroups={'person':0,'org':1,'contract':2,'date':3,'money':4,'location':5};
var chart=echarts.init(document.getElementById('graphChart'));
var currentGraph={nodes:[],edges:[]};
var allEntities=[],allRelations=[],currentFocus=null;

function showPlaceholder(){
  chart.setOption({
    backgroundColor:'transparent',
    graphic:{type:'text',left:'center',top:'center',style:{text:'上传文档以构建知识图谱',font:'16px "Microsoft YaHei",sans-serif',fill:'#64748b',textAlign:'center'}}
  });
}
showPlaceholder();

document.getElementById('dropZone').ondragover=function(e){e.preventDefault()};
document.getElementById('dropZone').ondrop=function(e){e.preventDefault();var f=e.dataTransfer.files[0];if(f)doUpload(f)};
function uploadDoc(){var f=document.getElementById('fileInput').files[0];if(f)doUpload(f)}

async function doUpload(file){
  document.getElementById('uploadStatus').innerHTML='<span class="spinner"></span> 解析中...';
  var fd=new FormData();fd.append('file',file);
  try{
    var r=await fetch(API+'/build',{method:'POST',body:fd});var d=await r.json();
    if(d.code===0){
      allEntities=d.data.entities||[];allRelations=d.data.relations||[];
      toast('提取 '+d.data.entity_count+' 个实体, '+d.data.relation_count+' 条关系','success');
      renderEntities(allEntities);
      renderRelations(allRelations);
      if(allEntities.length>0){setTimeout(function(){focusEntity(allEntities[0].id,allEntities[0].name)},300)}
    }else{toast(d.message,'error')}
  }catch(e){toast(e.message,'error')}
  document.getElementById('uploadStatus').innerHTML='';
}

function renderEntities(entities){
  document.getElementById('entityList').innerHTML=entities.map(function(e){
    return '<div class="entity-item" onclick="focusEntity(\''+e.id+'\',\''+e.name+'\')" style="'+(currentFocus===e.id?'border-color:var(--accent)':'')+'"><span class="type-badge" style="background:'+(typeColors[e.type]||'#666')+'">'+(typeLabels[e.type]||e.type)+'</span><span style="font-size:13px">'+e.name+'</span><br><span style="font-size:11px;color:var(--muted)">来源: '+e.doc_name+'</span></div>';
  }).join('');
}

function renderRelations(relations){
  document.getElementById('relationList').innerHTML=relations.slice(0,20).map(function(r){
    return '<div class="relation-item">'+r.predicate+' <span style="color:var(--muted)">|</span> '+r.evidence.substring(0,40)+'</div>';
  }).join('');
}

function updateGraph(entities,relations){
  currentGraph={
    nodes:entities.map(function(e){return{id:e.id,name:e.name,type:e.type,symbolSize:30,category:entityGroups[e.type]||0,itemStyle:{color:typeColors[e.type]||'#666'}}}),
    edges:relations.map(function(r){return{source:r.subject,target:r.object,label:{show:true,formatter:r.predicate,fontSize:10}}})
  };
  renderGraph();
}

var categories=Object.keys(typeLabels).map(function(k,i){return{name:typeLabels[k],itemStyle:{color:Object.values(typeColors)[i]}}});

function renderGraph(){
  chart.setOption({
    backgroundColor:'transparent',
    tooltip:{formatter:function(p){return p.dataType==='node'?p.name+'<br>'+typeLabels[p.data.type]:p.name}},
    legend:{show:true,textStyle:{color:'#94a3b8'},data:categories.map(function(c){return c.name})},
    series:[{
      type:'graph',layout:'force',roam:true,draggable:true,
      force:{repulsion:400,edgeLength:[80,250],gravity:0.1},
      data:currentGraph.nodes,
      edges:currentGraph.edges,
      categories:categories,
      label:{show:true,position:'right',fontSize:12,color:'#e2e8f0'},
      lineStyle:{color:'#475569',curveness:0.2,width:1}
    }]
  });
}

async function searchEntity(){
  var name=document.getElementById('searchInput').value.trim();
  if(!name)return;
  try{
    var r=await fetch(API+'/search?name='+encodeURIComponent(name));var d=await r.json();
    if(d.code===0&&d.data&&d.data.length>0){
      renderEntities(d.data);
      if(d.data.length===1)focusEntity(d.data[0].id,d.data[0].name);
    }else{toast('未找到实体','error')}
  }catch(e){toast(e.message,'error')}
}

async function focusEntity(entityID,name){
  currentFocus=entityID;
  try{
    var r=await fetch(API+'/graph?entity_id='+entityID+'&depth=2');var d=await r.json();
    if(d.code===0&&d.data){
      currentGraph={nodes:d.data.nodes||[],edges:d.data.edges||[]};
      renderGraph();
      renderEntities(allEntities);
      toast('聚焦: '+name+' (节点:'+currentGraph.nodes.length+', 边:'+currentGraph.edges.length+')','success');
    }
  }catch(e){toast(e.message,'error')}
}

function toast(msg,type){
  var e=document.getElementById('toast'),d=document.createElement('div');
  d.className='toast-msg toast-'+type;d.textContent=msg;e.appendChild(d);
  setTimeout(function(){d.remove()},3000);
}
</script></body></html>`
